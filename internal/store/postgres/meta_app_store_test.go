//go:build integration

package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/webitel/im-providers-service/internal/domain/model"
	"github.com/webitel/im-providers-service/internal/store"
)

func newMetaAppStore() store.MetaAppStore {
	return NewMetaAppStore(testPool, testCrypto)
}

func insertTestMetaApp(t *testing.T, s store.MetaAppStore, a *model.MetaApp) *model.MetaApp {
	t.Helper()
	if err := s.Insert(context.Background(), a); err != nil {
		t.Fatalf("Insert MetaApp: %v", err)
	}
	t.Cleanup(func() {
		_ = s.Delete(context.Background(), a.ID)
	})
	return a
}

func TestMetaAppStore_Insert(t *testing.T) {
	s := newMetaAppStore()
	a := &model.MetaApp{
		URI:              "test-uri-insert",
		Name:             "Test App",
		AppID:            "fb-app-id",
		AppSecret:        "my-secret",
		OAuthRedirectURI: "https://example.com/callback",
		Scopes:           []string{"pages_messaging"},
		VerifyToken:      "verify-tok",
	}
	insertTestMetaApp(t, s, a)

	if a.ID == "" {
		t.Error("expected ID to be populated after Insert")
	}
	if a.AppSecret != "my-secret" {
		t.Errorf("expected decrypted secret after Insert, got %q", a.AppSecret)
	}
	if a.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestMetaAppStore_Select(t *testing.T) {
	s := newMetaAppStore()
	a := &model.MetaApp{
		URI:         "test-uri-select",
		Name:        "Select App",
		AppID:       "app-select",
		AppSecret:   "sec-select",
		Scopes:      []string{"pages_read_engagement"},
		VerifyToken: "vt",
	}
	insertTestMetaApp(t, s, a)

	got, err := s.Select(context.Background(), a.ID)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if got.ID != a.ID {
		t.Errorf("id mismatch: got %s want %s", got.ID, a.ID)
	}
	if got.AppSecret != "sec-select" {
		t.Errorf("expected decrypted secret, got %q", got.AppSecret)
	}
	if got.Name != "Select App" {
		t.Errorf("name mismatch: got %s", got.Name)
	}
}

func TestMetaAppStore_Select_NotFound(t *testing.T) {
	s := newMetaAppStore()
	_, err := s.Select(context.Background(), "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMetaAppStore_SelectByURI(t *testing.T) {
	s := newMetaAppStore()
	a := &model.MetaApp{
		URI:         "test-uri-by-uri",
		Name:        "URI App",
		AppID:       "app-byuri",
		AppSecret:   "sec-byuri",
		VerifyToken: "vt2",
		Scopes:      []string{},
	}
	insertTestMetaApp(t, s, a)

	got, err := s.SelectByURI(context.Background(), "test-uri-by-uri")
	if err != nil {
		t.Fatalf("SelectByURI: %v", err)
	}
	if got.URI != "test-uri-by-uri" {
		t.Errorf("unexpected URI: %s", got.URI)
	}
	if got.AppSecret != "sec-byuri" {
		t.Errorf("expected decrypted secret, got %q", got.AppSecret)
	}
}

func TestMetaAppStore_SelectByURI_NotFound(t *testing.T) {
	s := newMetaAppStore()
	_, err := s.SelectByURI(context.Background(), "no-such-uri")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMetaAppStore_Update(t *testing.T) {
	s := newMetaAppStore()
	a := &model.MetaApp{
		URI:         "test-uri-update",
		Name:        "Old Name",
		AppID:       "app-upd",
		AppSecret:   "old-sec",
		VerifyToken: "old-vt",
		Scopes:      []string{"pages_messaging"},
	}
	insertTestMetaApp(t, s, a)

	a.Name = "New Name"
	a.AppSecret = "new-sec"
	a.VerifyToken = "new-vt"
	a.Scopes = []string{"pages_messaging", "pages_read_engagement"}

	if err := s.Update(context.Background(), a); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if a.AppSecret != "new-sec" {
		t.Errorf("expected decrypted secret after Update, got %q", a.AppSecret)
	}

	got, err := s.Select(context.Background(), a.ID)
	if err != nil {
		t.Fatalf("Select after Update: %v", err)
	}
	if got.Name != "New Name" {
		t.Errorf("name not persisted: %s", got.Name)
	}
	if got.AppSecret != "new-sec" {
		t.Errorf("secret not persisted: %s", got.AppSecret)
	}
	if len(got.Scopes) != 2 {
		t.Errorf("scopes not persisted: %v", got.Scopes)
	}
}

func TestMetaAppStore_Delete(t *testing.T) {
	s := newMetaAppStore()
	a := &model.MetaApp{
		URI:         "test-uri-delete",
		Name:        "Delete App",
		AppID:       "app-del",
		AppSecret:   "sec-del",
		VerifyToken: "vt-del",
		Scopes:      []string{},
	}
	if err := s.Insert(context.Background(), a); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	if err := s.Delete(context.Background(), a.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := s.Select(context.Background(), a.ID)
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound after Delete, got %v", err)
	}
}

func TestMetaAppStore_Delete_NotFound(t *testing.T) {
	s := newMetaAppStore()
	err := s.Delete(context.Background(), "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
