//go:build integration

package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/webitel/im-providers-service/internal/domain/model"
	"github.com/webitel/im-providers-service/internal/store"
)

func newFBStore() store.FacebookStore {
	return NewFacebookStore(testPool, testCrypto, testCache)
}

// insertTestApp inserts a MetaApp and registers cleanup.
func insertTestApp(t *testing.T, uri string) *model.MetaApp {
	t.Helper()
	ms := newMetaAppStore()
	a := &model.MetaApp{
		URI:         uri,
		Name:        "FB Test App",
		AppID:       "fb-app-" + uri,
		AppSecret:   "test-secret",
		VerifyToken: "vt",
		Scopes:      []string{},
	}
	if err := ms.Insert(context.Background(), a); err != nil {
		t.Fatalf("insertTestApp Insert: %v", err)
	}
	t.Cleanup(func() { _ = ms.Delete(context.Background(), a.ID) })
	return a
}

func buildFBGate(appID, pageID string) *model.FacebookGate {
	return &model.FacebookGate{
		Name:      "FB Gate " + pageID,
		MetaAppID: appID,
		PageID:    pageID,
		PageToken: "raw-token",
		Enabled:   true,
		Peer: model.Peer{
			Sub: "sub-" + pageID,
			Iss: "iss-" + pageID,
		},
	}
}

func insertTestFBGate(t *testing.T, s store.FacebookStore, dc int64, g *model.FacebookGate) {
	t.Helper()
	if err := s.Insert(context.Background(), dc, g); err != nil {
		t.Fatalf("Insert FacebookGate: %v", err)
	}
	t.Cleanup(func() {
		_ = s.Unbind(context.Background(), g.ID)
	})
}

func TestFacebookStore_Insert(t *testing.T) {
	s := newFBStore()
	app := insertTestApp(t, "fb-uri-insert")
	g := buildFBGate(app.ID, "page-ins-1")

	insertTestFBGate(t, s, 10, g)

	if g.ID == "" {
		t.Error("expected gate ID to be populated after Insert")
	}
	if g.PageName != g.Name {
		t.Errorf("expected PageName=%s, got %s", g.Name, g.PageName)
	}
	if g.Status != model.StatusActive {
		t.Errorf("expected StatusActive, got %v", g.Status)
	}
	if g.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}

	// Cache should contain the gate state.
	state, ok := testCache.Get(g.PageID)
	if !ok {
		t.Error("expected gate state in cache after Insert")
	}
	if state.GateID != g.ID {
		t.Errorf("cache GateID mismatch: got %s want %s", state.GateID, g.ID)
	}
}

func TestFacebookStore_Select(t *testing.T) {
	s := newFBStore()
	app := insertTestApp(t, "fb-uri-select")
	g := buildFBGate(app.ID, "page-sel-1")
	insertTestFBGate(t, s, 11, g)

	got, err := s.Select(context.Background(), g.ID)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if got.ID != g.ID {
		t.Errorf("ID mismatch: got %s want %s", got.ID, g.ID)
	}
	if got.PageToken != "raw-token" {
		t.Errorf("expected decrypted page token, got %q", got.PageToken)
	}
	if got.PageID != g.PageID {
		t.Errorf("PageID mismatch: got %s want %s", got.PageID, g.PageID)
	}
	if got.Peer.Sub != g.Peer.Sub {
		t.Errorf("Peer.Sub mismatch: got %s want %s", got.Peer.Sub, g.Peer.Sub)
	}
}

func TestFacebookStore_Select_NotFound(t *testing.T) {
	s := newFBStore()
	_, err := s.Select(context.Background(), "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestFacebookStore_SelectByPageAndURI(t *testing.T) {
	s := newFBStore()
	app := insertTestApp(t, "fb-uri-bypage")
	g := buildFBGate(app.ID, "page-bypage-1")
	insertTestFBGate(t, s, 12, g)

	got, err := s.SelectByPageAndURI(context.Background(), g.PageID, app.URI)
	if err != nil {
		t.Fatalf("SelectByPageAndURI: %v", err)
	}
	if got.ID != g.ID {
		t.Errorf("ID mismatch: got %s want %s", got.ID, g.ID)
	}
	if got.PageToken != "raw-token" {
		t.Errorf("expected decrypted token, got %q", got.PageToken)
	}
}

func TestFacebookStore_SelectByPageAndURI_NotFound(t *testing.T) {
	s := newFBStore()
	_, err := s.SelectByPageAndURI(context.Background(), "no-such-page", "no-such-uri")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestFacebookStore_Update(t *testing.T) {
	s := newFBStore()
	app := insertTestApp(t, "fb-uri-update")
	g := buildFBGate(app.ID, "page-upd-1")
	insertTestFBGate(t, s, 13, g)

	g.Name = "Updated Gate"
	g.PageToken = "new-token"
	g.Enabled = false

	if err := s.Update(context.Background(), g); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := s.Select(context.Background(), g.ID)
	if err != nil {
		t.Fatalf("Select after Update: %v", err)
	}
	if got.Name != "Updated Gate" {
		t.Errorf("name not persisted: %s", got.Name)
	}
	if got.PageToken != "new-token" {
		t.Errorf("token not persisted: %s", got.PageToken)
	}
	if got.Status != model.StatusDisabled {
		t.Errorf("expected StatusDisabled after update, got %v", got.Status)
	}
}

func TestFacebookStore_Unbind(t *testing.T) {
	s := newFBStore()
	app := insertTestApp(t, "fb-uri-unbind")
	g := buildFBGate(app.ID, "page-unb-1")

	if err := s.Insert(context.Background(), 14, g); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	if err := s.Unbind(context.Background(), g.ID); err != nil {
		t.Fatalf("Unbind: %v", err)
	}

	_, err := s.Select(context.Background(), g.ID)
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound after Unbind, got %v", err)
	}

	// Cache entry should be cleared.
	_, ok := testCache.Get(g.PageID)
	if ok {
		t.Error("expected cache entry to be cleared after Unbind")
	}
}

func TestFacebookStore_Unbind_NotFound(t *testing.T) {
	s := newFBStore()
	err := s.Unbind(context.Background(), "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
