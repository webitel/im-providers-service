//go:build integration

package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/webitel/im-providers-service/internal/domain/model"
	"github.com/webitel/im-providers-service/internal/store"
)

func newGateStore() store.GateStore {
	return NewGateStore(testPool, testCfg)
}

// insertGateForList creates a Facebook gate so that gate_summary has rows.
// Returns the gate ID and a cleanup function.
func insertGateForList(t *testing.T, uri, pageID string) string {
	t.Helper()
	app := insertTestApp(t, uri)
	g := buildFBGate(app.ID, pageID)
	fs := newFBStore()
	if err := fs.Insert(context.Background(), 99, g); err != nil {
		t.Fatalf("insertGateForList Insert: %v", err)
	}
	t.Cleanup(func() { _ = fs.Unbind(context.Background(), g.ID) })
	return g.ID
}

func TestGateStore_List_Empty(t *testing.T) {
	// Delete all existing rows so we start clean. We rely on other tests having
	// their own cleanup — run this as the very first sub-test via a unique
	// filter that matches nothing instead of wiping global state.
	gs := newGateStore()
	list, next, err := gs.List(context.Background(), model.ListFilter{Page: 9999, Size: 100})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0 results for page 9999, got %d", len(list))
	}
	if next {
		t.Error("expected next=false")
	}
}

func TestGateStore_List_ReturnsGates(t *testing.T) {
	gs := newGateStore()
	insertGateForList(t, "gs-uri-list-1", "gs-page-list-1")
	insertGateForList(t, "gs-uri-list-2", "gs-page-list-2")

	list, _, err := gs.List(context.Background(), model.ListFilter{Page: 0, Size: 100})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) < 2 {
		t.Errorf("expected at least 2 gates, got %d", len(list))
	}
	// Each row should have an ID and a known type.
	for _, g := range list {
		if g.ID == "" {
			t.Error("gate has empty ID")
		}
	}
}

func TestGateStore_List_NextPage(t *testing.T) {
	gs := newGateStore()
	// Insert 3 gates, request size=2 → next should be true.
	insertGateForList(t, "gs-uri-np-1", "gs-page-np-1")
	insertGateForList(t, "gs-uri-np-2", "gs-page-np-2")
	insertGateForList(t, "gs-uri-np-3", "gs-page-np-3")

	list, next, err := gs.List(context.Background(), model.ListFilter{Page: 0, Size: 2})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected exactly 2 results (size=2), got %d", len(list))
	}
	if !next {
		t.Error("expected next=true when more rows exist beyond the page")
	}
}

func TestGateStore_Delete(t *testing.T) {
	gs := newGateStore()
	// Manually insert and then delete via GateStore.Delete.
	app := insertTestApp(t, "gs-uri-del")
	g := buildFBGate(app.ID, "gs-page-del-1")
	fs := newFBStore()
	if err := fs.Insert(context.Background(), 20, g); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	if err := gs.Delete(context.Background(), g.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// FacebookStore.Select should now return ErrNotFound.
	_, err := fs.Select(context.Background(), g.ID)
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound after Delete, got %v", err)
	}
}

func TestGateStore_Delete_NotFound(t *testing.T) {
	gs := newGateStore()
	err := gs.Delete(context.Background(), "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
