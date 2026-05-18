package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/webitel/im-providers-service/internal/domain/model"
	"github.com/webitel/im-providers-service/internal/store"
)

var noopLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

// -- mock store --

type mockFacebookStore struct {
	insertFn            func(ctx context.Context, dc int64, g *model.FacebookGate) error
	selectFn            func(ctx context.Context, id string) (*model.FacebookGate, error)
	selectByPageAndURIFn func(ctx context.Context, pageID, uri string) (*model.FacebookGate, error)
	updateFn            func(ctx context.Context, g *model.FacebookGate) error
	unbindFn            func(ctx context.Context, gateID string) error
}

func (m *mockFacebookStore) Insert(ctx context.Context, dc int64, g *model.FacebookGate) error {
	return m.insertFn(ctx, dc, g)
}
func (m *mockFacebookStore) Select(ctx context.Context, id string) (*model.FacebookGate, error) {
	return m.selectFn(ctx, id)
}
func (m *mockFacebookStore) SelectByPageAndURI(ctx context.Context, pageID, uri string) (*model.FacebookGate, error) {
	return m.selectByPageAndURIFn(ctx, pageID, uri)
}
func (m *mockFacebookStore) Update(ctx context.Context, g *model.FacebookGate) error {
	return m.updateFn(ctx, g)
}
func (m *mockFacebookStore) Unbind(ctx context.Context, gateID string) error {
	return m.unbindFn(ctx, gateID)
}

func stubFBGate() *model.FacebookGate {
	return &model.FacebookGate{
		ID:        "gate-1",
		Name:      "Test Page",
		MetaAppID: "app-1",
		PageID:    "page-1",
		PageToken: "tok",
		Enabled:   true,
		Status:    model.StatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func newFBService(repo store.FacebookStore) *FacebookService {
	return NewFacebookService(repo, nil, noopLogger)
}

// -- tests --

func TestFacebookService_CreateGate_Success(t *testing.T) {
	var inserted *model.FacebookGate
	repo := &mockFacebookStore{
		insertFn: func(_ context.Context, dc int64, g *model.FacebookGate) error {
			if dc != 7 {
				t.Errorf("unexpected dc: %d", dc)
			}
			g.ID = "gate-1"
			inserted = g
			return nil
		},
	}
	svc := newFBService(repo)
	gate, err := svc.CreateGate(context.Background(), model.CreateFacebook{
		Name:      "Test Page",
		Dc:        7,
		MetaAppID: "app-1",
		PageID:    "page-1",
		PageToken: "tok",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gate.ID != "gate-1" {
		t.Errorf("unexpected gate id: %s", gate.ID)
	}
	if !inserted.Enabled {
		t.Error("gate should be enabled after creation")
	}
}

func TestFacebookService_CreateGate_StoreError(t *testing.T) {
	repo := &mockFacebookStore{
		insertFn: func(_ context.Context, _ int64, _ *model.FacebookGate) error {
			return errors.New("db error")
		},
	}
	svc := newFBService(repo)
	_, err := svc.CreateGate(context.Background(), model.CreateFacebook{Name: "X"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFacebookService_GetGate_Success(t *testing.T) {
	repo := &mockFacebookStore{
		selectFn: func(_ context.Context, id string) (*model.FacebookGate, error) {
			if id != "gate-1" {
				t.Errorf("unexpected id: %s", id)
			}
			return stubFBGate(), nil
		},
	}
	svc := newFBService(repo)
	gate, err := svc.GetGate(context.Background(), "gate-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gate.ID != "gate-1" {
		t.Errorf("unexpected gate id: %s", gate.ID)
	}
}

func TestFacebookService_GetGate_NotFound(t *testing.T) {
	repo := &mockFacebookStore{
		selectFn: func(_ context.Context, _ string) (*model.FacebookGate, error) {
			return nil, store.ErrNotFound
		},
	}
	svc := newFBService(repo)
	_, err := svc.GetGate(context.Background(), "missing")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestFacebookService_UpdateGate_Name(t *testing.T) {
	repo := &mockFacebookStore{
		selectFn: func(_ context.Context, _ string) (*model.FacebookGate, error) {
			return stubFBGate(), nil
		},
		updateFn: func(_ context.Context, g *model.FacebookGate) error {
			if g.Name != "New Name" {
				t.Errorf("unexpected name: %s", g.Name)
			}
			return nil
		},
	}
	svc := newFBService(repo)
	newName := "New Name"
	gate, err := svc.UpdateGate(context.Background(), model.UpdateFacebook{
		ID:   "gate-1",
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gate.Name != "New Name" {
		t.Errorf("unexpected name: %s", gate.Name)
	}
}

func TestFacebookService_UpdateGate_Enabled(t *testing.T) {
	repo := &mockFacebookStore{
		selectFn: func(_ context.Context, _ string) (*model.FacebookGate, error) {
			g := stubFBGate()
			g.Enabled = true
			return g, nil
		},
		updateFn: func(_ context.Context, g *model.FacebookGate) error {
			if g.Enabled {
				t.Error("expected gate to be disabled")
			}
			return nil
		},
	}
	svc := newFBService(repo)
	disabled := false
	_, err := svc.UpdateGate(context.Background(), model.UpdateFacebook{
		ID:      "gate-1",
		Enabled: &disabled,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFacebookService_UpdateGate_SelectError(t *testing.T) {
	repo := &mockFacebookStore{
		selectFn: func(_ context.Context, _ string) (*model.FacebookGate, error) {
			return nil, store.ErrNotFound
		},
	}
	svc := newFBService(repo)
	name := "x"
	_, err := svc.UpdateGate(context.Background(), model.UpdateFacebook{ID: "missing", Name: &name})
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestFacebookService_UpdateGate_UpdateError(t *testing.T) {
	repo := &mockFacebookStore{
		selectFn: func(_ context.Context, _ string) (*model.FacebookGate, error) {
			return stubFBGate(), nil
		},
		updateFn: func(_ context.Context, _ *model.FacebookGate) error {
			return errors.New("update failed")
		},
	}
	svc := newFBService(repo)
	name := "x"
	_, err := svc.UpdateGate(context.Background(), model.UpdateFacebook{ID: "gate-1", Name: &name})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFacebookService_DeleteGate_Success(t *testing.T) {
	var unbound string
	repo := &mockFacebookStore{
		selectFn: func(_ context.Context, _ string) (*model.FacebookGate, error) {
			return stubFBGate(), nil
		},
		unbindFn: func(_ context.Context, id string) error {
			unbound = id
			return nil
		},
	}
	svc := newFBService(repo)
	gate, err := svc.DeleteGate(context.Background(), "gate-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gate.ID != "gate-1" {
		t.Errorf("unexpected gate id: %s", gate.ID)
	}
	if unbound != "gate-1" {
		t.Errorf("expected unbind for gate-1, got %s", unbound)
	}
}

func TestFacebookService_DeleteGate_NotFound(t *testing.T) {
	repo := &mockFacebookStore{
		selectFn: func(_ context.Context, _ string) (*model.FacebookGate, error) {
			return nil, store.ErrNotFound
		},
	}
	svc := newFBService(repo)
	_, err := svc.DeleteGate(context.Background(), "missing")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestFacebookService_DeleteGate_UnbindError(t *testing.T) {
	repo := &mockFacebookStore{
		selectFn: func(_ context.Context, _ string) (*model.FacebookGate, error) {
			return stubFBGate(), nil
		},
		unbindFn: func(_ context.Context, _ string) error {
			return errors.New("unbind failed")
		},
	}
	svc := newFBService(repo)
	_, err := svc.DeleteGate(context.Background(), "gate-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
