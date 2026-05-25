package postgres

import (
	"context"
	"fmt"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
	fbmodel "github.com/webitel/im-providers-service/internal/facebook/model"
	fbstore "github.com/webitel/im-providers-service/internal/facebook/store"
	"github.com/webitel/im-providers-service/pkg/crypto"
)

var _ fbstore.MetaAppStore = (*metaAppStore)(nil)

type metaAppStore struct {
	pool   *pgxpool.Pool
	crypto crypto.Encryptor
}

func NewMetaAppStore(pool *pgxpool.Pool, crypt crypto.Encryptor) fbstore.MetaAppStore {
	return &metaAppStore{pool: pool, crypto: crypt}
}

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

func (s *metaAppStore) decryptSecret(a *fbmodel.MetaApp) error {
	dec, err := s.crypto.Decrypt(a.AppSecret)
	if err != nil {
		return fmt.Errorf("crypto: decrypt app_secret: %w", err)
	}
	a.AppSecret = dec
	return nil
}
