package postgres

import (
	"context"
	"fmt"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/webitel/im-providers-service/internal/domain/model"
	"github.com/webitel/im-providers-service/internal/store"
	"github.com/webitel/im-providers-service/pkg/crypto"
)

// [INTERFACE GUARD]
var _ store.MetaAppStore = (*metaAppStore)(nil)

type metaAppStore struct {
	pool   *pgxpool.Pool
	crypto crypto.Encryptor
}

func NewMetaAppStore(pool *pgxpool.Pool, crypt crypto.Encryptor) store.MetaAppStore {
	return &metaAppStore{
		pool:   pool,
		crypto: crypt,
	}
}

// Insert registers a new Meta application and returns the fully populated struct.
func (s *metaAppStore) Insert(ctx context.Context, a *model.MetaApp) error {
	const query = `
		INSERT INTO im_provider.meta_apps (name, app_id, app_secret, redirect_uri, scopes)
		VALUES (@name, @app_id, @app_secret, @redirect_uri, @scopes)
		RETURNING *`

	secret, err := s.crypto.Encrypt(a.AppSecret)
	if err != nil {
		return fmt.Errorf("crypto: encrypt app_secret: %w", err)
	}

	args := pgx.NamedArgs{
		"name":         a.Name,
		"app_id":       a.AppID,
		"app_secret":   secret,
		"redirect_uri": a.OAuthRedirectURI,
		"scopes":       a.Scopes,
	}

	if err := pgxscan.Get(ctx, s.pool, a, query, args); err != nil {
		return err
	}

	return s.decryptSecret(a)
}

// Select finds app by ID and decrypts sensitive data.
func (s *metaAppStore) Select(ctx context.Context, id string) (*model.MetaApp, error) {
	const query = `SELECT * FROM im_provider.meta_apps WHERE id = $1`

	var a model.MetaApp
	if err := pgxscan.Get(ctx, s.pool, &a, query, id); err != nil {
		if pgxscan.NotFound(err) {
			return nil, store.ErrNotFound
		}
		return nil, err
	}

	if err := s.decryptSecret(&a); err != nil {
		return nil, err
	}
	return &a, nil
}

// Update updates the application settings and refreshes the struct state.
func (s *metaAppStore) Update(ctx context.Context, a *model.MetaApp) error {
	const query = `
		UPDATE im_provider.meta_apps 
		SET name = @name, 
		    app_secret = @app_secret, 
		    redirect_uri = @redirect_uri, 
		    scopes = @scopes, 
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
	}

	if err := pgxscan.Get(ctx, s.pool, a, query, args); err != nil {
		if pgxscan.NotFound(err) {
			return store.ErrNotFound
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
		return store.ErrNotFound
	}
	return nil
}

// decryptSecret is a private helper to decrypt the AppSecret.
func (s *metaAppStore) decryptSecret(a *model.MetaApp) error {
	dec, err := s.crypto.Decrypt(a.AppSecret)
	if err != nil {
		return fmt.Errorf("crypto: decrypt app_secret: %w", err)
	}
	a.AppSecret = dec
	return nil
}
