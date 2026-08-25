package service

import (
	"context"
	"errors"
	"testing"

	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
	igmodel "github.com/webitel/im-providers-service/internal/instagram/model"
	igstore "github.com/webitel/im-providers-service/internal/instagram/store"
)

// ---------------------------------------------------------------------------
// stub store
// ---------------------------------------------------------------------------

type mockInstagramStore struct {
	insertFn      func(ctx context.Context, dc int64, g *igmodel.InstagramGate) error
	selectFn      func(ctx context.Context, id string) (*igmodel.InstagramGate, error)
	updateFn      func(ctx context.Context, g *igmodel.InstagramGate) error
	unbindFn      func(ctx context.Context, gateID string) error
	selectByBizFn func(ctx context.Context, bizID, uri string) (*igmodel.InstagramGate, error)
}

func (m *mockInstagramStore) Insert(ctx context.Context, dc int64, g *igmodel.InstagramGate) error {
	if m.insertFn != nil {
		return m.insertFn(ctx, dc, g)
	}

	return nil
}

func (m *mockInstagramStore) Select(ctx context.Context, id string) (*igmodel.InstagramGate, error) {
	if m.selectFn != nil {
		return m.selectFn(ctx, id)
	}

	return nil, errors.New("not found")
}

func (m *mockInstagramStore) SelectByBusinessAccountAndURI(ctx context.Context, bizID, uri string) (*igmodel.InstagramGate, error) {
	if m.selectByBizFn != nil {
		return m.selectByBizFn(ctx, bizID, uri)
	}

	return nil, errors.New("not implemented")
}

func (m *mockInstagramStore) Update(ctx context.Context, g *igmodel.InstagramGate) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, g)
	}

	return nil
}

func (m *mockInstagramStore) Unbind(ctx context.Context, gateID string) error {
	if m.unbindFn != nil {
		return m.unbindFn(ctx, gateID)
	}

	return nil
}

var _ igstore.InstagramStore = (*mockInstagramStore)(nil)

func newSvc(store igstore.InstagramStore) *InstagramService {
	return NewInstagramService(store, noopLogger)
}

// ---------------------------------------------------------------------------
// CreateGate
// ---------------------------------------------------------------------------

func TestCreateGate_ValidationError_StopsInsert(t *testing.T) {
	insertCalled := false
	svc := newSvc(&mockInstagramStore{
		insertFn: func(_ context.Context, _ int64, _ *igmodel.InstagramGate) error {
			insertCalled = true

			return nil
		},
	})

	// Missing required fields.
	_, err := svc.CreateGate(context.Background(), igmodel.CreateInstagram{})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	var ve *igmodel.ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}

	if insertCalled {
		t.Error("Insert must not be called when validation fails")
	}
}

func TestCreateGate_InsertError_Propagates(t *testing.T) {
	dbErr := errors.New("db unavailable")
	svc := newSvc(&mockInstagramStore{
		insertFn: func(_ context.Context, _ int64, _ *igmodel.InstagramGate) error {
			return dbErr
		},
	})

	_, err := svc.CreateGate(context.Background(), igmodel.CreateInstagram{
		Name:              "My IG",
		MetaAppID:         "app-1",
		BusinessAccountID: "biz-1",
		IGToken:           "tok",
		Dc:                1,
	})
	if !errors.Is(err, dbErr) {
		t.Errorf("expected db error to propagate, got %v", err)
	}
}

func TestCreateGate_Success_EnabledTrue(t *testing.T) {
	var inserted *igmodel.InstagramGate

	svc := newSvc(&mockInstagramStore{
		insertFn: func(_ context.Context, _ int64, g *igmodel.InstagramGate) error {
			inserted = g

			return nil
		},
	})

	gate, err := svc.CreateGate(context.Background(), igmodel.CreateInstagram{
		Name:              "IG Gate",
		MetaAppID:         "app-1",
		BusinessAccountID: "biz-1",
		IGToken:           "token",
		Dc:                42,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gate == nil || !gate.Enabled {
		t.Error("created gate must be enabled by default")
	}

	if inserted == nil || inserted.Name != "IG Gate" {
		t.Error("insert must receive the populated gate")
	}
}

// ---------------------------------------------------------------------------
// GetGate
// ---------------------------------------------------------------------------

func TestGetGate_DelegatesToStore(t *testing.T) {
	want := &igmodel.InstagramGate{ID: "g-1", Name: "Test"}
	svc := newSvc(&mockInstagramStore{
		selectFn: func(_ context.Context, id string) (*igmodel.InstagramGate, error) {
			if id != "g-1" {
				t.Errorf("unexpected id: %s", id)
			}

			return want, nil
		},
	})

	got, err := svc.GetGate(context.Background(), "g-1", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.ID != "g-1" {
		t.Errorf("unexpected gate id: %s", got.ID)
	}
}

// ---------------------------------------------------------------------------
// UpdateGate
// ---------------------------------------------------------------------------

func TestUpdateGate_NotFound_Propagates(t *testing.T) {
	svc := newSvc(&mockInstagramStore{
		selectFn: func(_ context.Context, _ string) (*igmodel.InstagramGate, error) {
			return nil, errors.New("not found")
		},
	})

	_, err := svc.UpdateGate(context.Background(), igmodel.UpdateInstagram{ID: "missing"})
	if err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestUpdateGate_ApplyToCalledBeforeUpdate(t *testing.T) {
	original := &igmodel.InstagramGate{ID: "g-1", Name: "Old Name", Enabled: false}
	newName := "New Name"
	enabled := true

	var savedGate *igmodel.InstagramGate

	svc := newSvc(&mockInstagramStore{
		selectFn: func(_ context.Context, _ string) (*igmodel.InstagramGate, error) {
			return original, nil
		},
		updateFn: func(_ context.Context, g *igmodel.InstagramGate) error {
			savedGate = g

			return nil
		},
	})

	_, err := svc.UpdateGate(context.Background(), igmodel.UpdateInstagram{
		ID:      "g-1",
		Name:    &newName,
		Enabled: &enabled,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if savedGate == nil || savedGate.Name != "New Name" {
		t.Errorf("name not applied before update, got %v", savedGate)
	}

	if !savedGate.Enabled {
		t.Error("enabled flag not applied before update")
	}
}

func TestUpdateGate_UpdateError_Propagates(t *testing.T) {
	dbErr := errors.New("update failed")
	svc := newSvc(&mockInstagramStore{
		selectFn: func(_ context.Context, _ string) (*igmodel.InstagramGate, error) {
			return &igmodel.InstagramGate{ID: "g-1"}, nil
		},
		updateFn: func(_ context.Context, _ *igmodel.InstagramGate) error {
			return dbErr
		},
	})

	_, err := svc.UpdateGate(context.Background(), igmodel.UpdateInstagram{ID: "g-1"})
	if !errors.Is(err, dbErr) {
		t.Errorf("expected db error to propagate, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// DeleteGate
// ---------------------------------------------------------------------------

func TestDeleteGate_NotFound_Propagates(t *testing.T) {
	svc := newSvc(&mockInstagramStore{
		selectFn: func(_ context.Context, _ string) (*igmodel.InstagramGate, error) {
			return nil, errors.New("not found")
		},
	})

	_, err := svc.DeleteGate(context.Background(), "missing", 0)
	if err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestDeleteGate_UnbindCalled(t *testing.T) {
	unbindCalled := false
	svc := newSvc(&mockInstagramStore{
		selectFn: func(_ context.Context, id string) (*igmodel.InstagramGate, error) {
			return &igmodel.InstagramGate{ID: id, BusinessAccountID: "biz-1"}, nil
		},
		unbindFn: func(_ context.Context, gateID string) error {
			unbindCalled = true

			if gateID != "g-1" {
				return errors.New("unexpected gate id: " + gateID)
			}

			return nil
		},
	})

	gate, err := svc.DeleteGate(context.Background(), "g-1", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !unbindCalled {
		t.Error("Unbind must be called on delete")
	}

	if gate.ID != "g-1" {
		t.Errorf("expected returned gate id g-1, got %s", gate.ID)
	}
}

func TestDeleteGate_UnbindError_Propagates(t *testing.T) {
	dbErr := errors.New("unbind failed")
	svc := newSvc(&mockInstagramStore{
		selectFn: func(_ context.Context, id string) (*igmodel.InstagramGate, error) {
			return &igmodel.InstagramGate{ID: id}, nil
		},
		unbindFn: func(_ context.Context, _ string) error {
			return dbErr
		},
	})

	_, err := svc.DeleteGate(context.Background(), "g-1", 0)
	if !errors.Is(err, dbErr) {
		t.Errorf("expected unbind error to propagate, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// tenant isolation (authz) — a gate must only be reachable by its owning domain
// ---------------------------------------------------------------------------

func TestGetGate_CrossTenant_NotFound(t *testing.T) {
	svc := newSvc(&mockInstagramStore{
		selectFn: func(_ context.Context, _ string) (*igmodel.InstagramGate, error) {
			return &igmodel.InstagramGate{ID: "g-1", DomainID: 100}, nil
		},
	})

	_, err := svc.GetGate(context.Background(), "g-1", 200)
	if !errors.Is(err, sharedstore.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for another tenant's gate, got %v", err)
	}
}

func TestUpdateGate_CrossTenant_NotFound(t *testing.T) {
	updateCalled := false
	svc := newSvc(&mockInstagramStore{
		selectFn: func(_ context.Context, _ string) (*igmodel.InstagramGate, error) {
			return &igmodel.InstagramGate{ID: "g-1", DomainID: 100}, nil
		},
		updateFn: func(_ context.Context, _ *igmodel.InstagramGate) error {
			updateCalled = true

			return nil
		},
	})

	_, err := svc.UpdateGate(context.Background(), igmodel.UpdateInstagram{ID: "g-1", Dc: 200})
	if !errors.Is(err, sharedstore.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if updateCalled {
		t.Error("update must not run for another tenant's gate")
	}
}

func TestDeleteGate_CrossTenant_NotFound(t *testing.T) {
	unbindCalled := false
	svc := newSvc(&mockInstagramStore{
		selectFn: func(_ context.Context, _ string) (*igmodel.InstagramGate, error) {
			return &igmodel.InstagramGate{ID: "g-1", DomainID: 100}, nil
		},
		unbindFn: func(_ context.Context, _ string) error {
			unbindCalled = true

			return nil
		},
	})

	_, err := svc.DeleteGate(context.Background(), "g-1", 200)
	if !errors.Is(err, sharedstore.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if unbindCalled {
		t.Error("unbind must not run for another tenant's gate")
	}
}
