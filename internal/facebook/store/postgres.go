package store

import (
	"context"
	"fmt"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	fbmodel "github.com/webitel/im-providers-service/internal/facebook/model"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
	"github.com/webitel/im-providers-service/pkg/crypto"
)

// --- FacebookStore postgres implementation ---

// [INTERFACE GUARD]
var _ FacebookStore = (*facebookStore)(nil)

type facebookStore struct {
	pool   *pgxpool.Pool
	crypto crypto.Encryptor
	cache  sharedstore.GateCache
}

// NewFacebookStore initializes a storage provider for Facebook gateways.
func NewFacebookStore(pool *pgxpool.Pool, crypt crypto.Encryptor, cache sharedstore.GateCache) FacebookStore {
	return &facebookStore{
		pool:   pool,
		crypto: crypt,
		cache:  cache,
	}
}

// Insert creates a new gate linked to a domain (dc), a bot identity, and Facebook configurations.
func (s *facebookStore) Insert(ctx context.Context, dc int64, g *fbmodel.FacebookGate) error {
	token, err := s.crypto.Encrypt(g.PageToken)
	if err != nil {
		return fmt.Errorf("crypto: %w", err)
	}

	const query = `
	WITH new_gate AS (
		INSERT INTO im_provider.gates (dc, name, type, enabled)
		VALUES ($1, $2, 'facebook', $3)
		RETURNING id, name, created_at, updated_at
	),
	new_bot AS (
		INSERT INTO im_provider.bots (sub, iss, gate_id)
		SELECT $4, $5, id FROM new_gate
		RETURNING id
	)
	INSERT INTO im_provider.facebook (gate_id, meta_app_id, page_id, page_token)
	SELECT id, $6, $7, $8 FROM new_gate
	RETURNING
		gate_id,
		(SELECT name FROM new_gate),
		(SELECT created_at FROM new_gate),
		(SELECT updated_at FROM new_gate)`

	err = s.pool.QueryRow(ctx, query,
		dc, g.Name, g.Enabled, g.Peer.Sub, g.Peer.Iss, g.MetaAppID, g.PageID, token,
	).Scan(&g.ID, &g.Name, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return fmt.Errorf("postgres: insert facebook gateway: %w", err)
	}

	// Cache GateState including Issuer for routing
	s.cache.Set(g.PageID, sharedstore.GateState{
		GateID:  g.ID,
		Enabled: g.Enabled,
		Issuer:  g.Peer.Iss,
		Sub:     g.Peer.Sub,
	})

	s.mapVirtualFields(g)
	return nil
}

// Select fetches a single Facebook gateway with its peer identity by gate UUID.
func (s *facebookStore) Select(ctx context.Context, id string) (*fbmodel.FacebookGate, error) {
	const query = `
	SELECT
		g.id, g.name, g.enabled, g.created_at, g.updated_at,
		b.sub AS "peer.sub", b.iss AS "peer.iss",
		fb.meta_app_id, fb.page_id, fb.page_token
	FROM im_provider.gates g
	JOIN im_provider.bots b ON g.id = b.gate_id
	JOIN im_provider.facebook fb ON g.id = fb.gate_id
	WHERE g.id = $1`

	var g fbmodel.FacebookGate
	if err := pgxscan.Get(ctx, s.pool, &g, query, id); err != nil {
		if pgxscan.NotFound(err) {
			return nil, sharedstore.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: select facebook gate: %w", err)
	}

	if dec, err := s.crypto.Decrypt(g.PageToken); err == nil {
		g.PageToken = dec
	}

	s.mapVirtualFields(&g)
	return &g, nil
}

// SelectByPageAndURI joins with meta_apps to filter by the unique URI slug.
func (s *facebookStore) SelectByPageAndURI(ctx context.Context, pageID, uri string) (*fbmodel.FacebookGate, error) {
	const query = `
	SELECT
		g.id,
		g.dc AS domain_id,
		g.name,
		g.enabled,
		g.created_at,
		g.updated_at,
		b.sub AS "peer.sub",
		b.iss AS "peer.iss",
		fb.meta_app_id,
		fb.page_id,
		fb.page_token
	FROM im_provider.gates g
	JOIN im_provider.bots b ON g.id = b.gate_id
	JOIN im_provider.facebook fb ON g.id = fb.gate_id
	JOIN im_provider.meta_apps ma ON fb.meta_app_id = ma.id
	WHERE fb.page_id = $1 AND ma.uri = $2`

	var g fbmodel.FacebookGate
	if err := pgxscan.Get(ctx, s.pool, &g, query, pageID, uri); err != nil {
		if pgxscan.NotFound(err) {
			return nil, sharedstore.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: select by page_id and uri: %w", err)
	}

	if dec, err := s.crypto.Decrypt(g.PageToken); err == nil {
		g.PageToken = dec
	}

	s.mapVirtualFields(&g)

	return &g, nil
}

// Update modifies core metadata, identity, and credentials.
func (s *facebookStore) Update(ctx context.Context, g *fbmodel.FacebookGate) error {
	token, err := s.crypto.Encrypt(g.PageToken)
	if err != nil {
		return fmt.Errorf("crypto: %w", err)
	}

	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		// Update base gate.
		const uGate = `UPDATE im_provider.gates SET name = $1, enabled = $2, updated_at = NOW() WHERE id = $3 RETURNING updated_at`
		if err := tx.QueryRow(ctx, uGate, g.Name, g.Enabled, g.ID).Scan(&g.UpdatedAt); err != nil {
			return err
		}

		// Update identity (peer).
		const uBot = `UPDATE im_provider.bots SET sub = $1, iss = $2 WHERE gate_id = $3`
		if _, err := tx.Exec(ctx, uBot, g.Peer.Sub, g.Peer.Iss, g.ID); err != nil {
			return err
		}

		// Update Facebook config.
		const uConfig = `
			UPDATE im_provider.facebook
			SET meta_app_id = $1, page_id = $2, page_token = $3
			WHERE gate_id = $4`
		_, err := tx.Exec(ctx, uConfig, g.MetaAppID, g.PageID, token, g.ID)
		return err
	})

	if err == nil {
		s.cache.Delete(g.PageID)
	}
	return err
}

// Unbind deletes the gate and invalidates cache.
func (s *facebookStore) Unbind(ctx context.Context, gateID string) error {
	var pageID string
	_ = s.pool.QueryRow(ctx, "SELECT page_id FROM im_provider.facebook WHERE gate_id = $1", gateID).Scan(&pageID)

	if pageID != "" {
		s.cache.Delete(pageID)
	}

	const query = `DELETE FROM im_provider.gates WHERE id = $1`
	res, err := s.pool.Exec(ctx, query, gateID)
	if err != nil {
		return fmt.Errorf("postgres: delete gate: %w", err)
	}
	if res.RowsAffected() == 0 {
		return sharedstore.ErrNotFound
	}
	return nil
}

// mapVirtualFields sets helper fields for the application layer.
func (s *facebookStore) mapVirtualFields(g *fbmodel.FacebookGate) {
	g.PageName = g.Name
	if g.Enabled {
		g.Status = sharedmodel.StatusActive
	} else {
		g.Status = sharedmodel.StatusDisabled
	}
}

// --- MetaAppStore postgres implementation ---

var _ MetaAppStore = (*metaAppStore)(nil)

type metaAppStore struct {
	pool   *pgxpool.Pool
	crypto crypto.Encryptor
}

func NewMetaAppStore(pool *pgxpool.Pool, crypt crypto.Encryptor) MetaAppStore {
	return &metaAppStore{
		pool:   pool,
		crypto: crypt,
	}
}

// Insert registers a new Meta application and returns the fully populated struct.
func (s *metaAppStore) Insert(ctx context.Context, a *fbmodel.MetaApp) error {
	const query = `
		INSERT INTO im_provider.meta_apps (name, uri, app_id, app_secret, redirect_uri, scopes, verify_token)
		VALUES (@name, @uri, @app_id, @app_secret, @redirect_uri, @scopes, @verify_token)
		RETURNING *`

	secret, err := s.crypto.Encrypt(a.AppSecret)
	if err != nil {
		return fmt.Errorf("crypto: encrypt app_secret: %w", err)
	}

	args := pgx.NamedArgs{
		"name":         a.Name,
		"uri":          a.URI,
		"app_id":       a.AppID,
		"app_secret":   secret,
		"redirect_uri": a.OAuthRedirectURI,
		"scopes":       a.Scopes,
		"verify_token": a.VerifyToken,
	}

	if err := pgxscan.Get(ctx, s.pool, a, query, args); err != nil {
		return err
	}

	return s.decryptSecret(a)
}

// Select finds app by ID and decrypts sensitive data.
func (s *metaAppStore) Select(ctx context.Context, id string) (*fbmodel.MetaApp, error) {
	const query = `SELECT * FROM im_provider.meta_apps WHERE id = $1`

	var a fbmodel.MetaApp
	if err := pgxscan.Get(ctx, s.pool, &a, query, id); err != nil {
		if pgxscan.NotFound(err) {
			return nil, sharedstore.ErrNotFound
		}
		return nil, err
	}

	if err := s.decryptSecret(&a); err != nil {
		return nil, err
	}
	return &a, nil
}

// SelectByURI finds a MetaApp by its URI slug.
func (s *metaAppStore) SelectByURI(ctx context.Context, uri string) (*fbmodel.MetaApp, error) {
	const query = `SELECT * FROM im_provider.meta_apps WHERE uri = $1`

	var a fbmodel.MetaApp
	if err := pgxscan.Get(ctx, s.pool, &a, query, uri); err != nil {
		if pgxscan.NotFound(err) {
			return nil, sharedstore.ErrNotFound
		}
		return nil, err
	}

	if err := s.decryptSecret(&a); err != nil {
		return nil, err
	}
	return &a, nil
}

// Update updates the application settings and refreshes the struct state.
func (s *metaAppStore) Update(ctx context.Context, a *fbmodel.MetaApp) error {
	const query = `
		UPDATE im_provider.meta_apps
		SET name = @name,
		    app_secret = @app_secret,
		    redirect_uri = @redirect_uri,
		    scopes = @scopes,
		    verify_token = @verify_token,
		    updated_at = NOW()
		WHERE id = @id
		RETURNING *`

	secret, err := s.crypto.Encrypt(a.AppSecret)
	if err != nil {
		return fmt.Errorf("crypto: encrypt app_secret: %w", err)
	}

	args := pgx.NamedArgs{
		"id":           a.ID,
		"name":         a.Name,
		"app_secret":   secret,
		"redirect_uri": a.OAuthRedirectURI,
		"scopes":       a.Scopes,
		"verify_token": a.VerifyToken,
	}

	if err := pgxscan.Get(ctx, s.pool, a, query, args); err != nil {
		if pgxscan.NotFound(err) {
			return sharedstore.ErrNotFound
		}
		return err
	}

	return s.decryptSecret(a)
}

// Delete removes the app record from the database.
func (s *metaAppStore) Delete(ctx context.Context, id string) error {
	const query = `DELETE FROM im_provider.meta_apps WHERE id = $1`

	res, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if res.RowsAffected() == 0 {
		return sharedstore.ErrNotFound
	}
	return nil
}

// decryptSecret is a private helper to decrypt the AppSecret.
func (s *metaAppStore) decryptSecret(a *fbmodel.MetaApp) error {
	dec, err := s.crypto.Decrypt(a.AppSecret)
	if err != nil {
		return fmt.Errorf("crypto: decrypt app_secret: %w", err)
	}
	a.AppSecret = dec
	return nil
}
