package handler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	"github.com/webitel/im-providers-service/infra/auth"
	fbmodel "github.com/webitel/im-providers-service/internal/facebook/model"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

var noopLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

// -- auth helpers --

type mockIdentity struct{ domainID int64 }

func (m *mockIdentity) GetContactID() string { return "contact-1" }
func (m *mockIdentity) GetDomainID() int64   { return m.domainID }
func (m *mockIdentity) GetName() string      { return "test-user" }

func ctxWithAuth(domainID int64) context.Context {
	return context.WithValue(context.Background(), auth.AuthContextKey, &mockIdentity{domainID: domainID})
}

// -- mock service --

type mockFacebookService struct {
	createFn func(ctx context.Context, req fbmodel.CreateFacebook) (*fbmodel.FacebookGate, error)
	getFn    func(ctx context.Context, id string) (*fbmodel.FacebookGate, error)
	updateFn func(ctx context.Context, req fbmodel.UpdateFacebook) (*fbmodel.FacebookGate, error)
	deleteFn func(ctx context.Context, id string) (*fbmodel.FacebookGate, error)
}

func (m *mockFacebookService) CreateGate(ctx context.Context, req fbmodel.CreateFacebook) (*fbmodel.FacebookGate, error) {
	return m.createFn(ctx, req)
}
func (m *mockFacebookService) GetGate(ctx context.Context, id string) (*fbmodel.FacebookGate, error) {
	return m.getFn(ctx, id)
}
func (m *mockFacebookService) UpdateGate(ctx context.Context, req fbmodel.UpdateFacebook) (*fbmodel.FacebookGate, error) {
	return m.updateFn(ctx, req)
}
func (m *mockFacebookService) DeleteGate(ctx context.Context, id string) (*fbmodel.FacebookGate, error) {
	return m.deleteFn(ctx, id)
}

func (m *mockFacebookService) SetPersistentMenu(_ context.Context, _ string, _ []fbmodel.MenuItem, _ bool) error {
	return nil
}
func (m *mockFacebookService) DeletePersistentMenu(_ context.Context, _ string) error { return nil }
func (m *mockFacebookService) SetGetStarted(_ context.Context, _ string, _ string) error {
	return nil
}
func (m *mockFacebookService) DeleteGetStarted(_ context.Context, _ string) error { return nil }

// -- helpers --

func stubGate() *fbmodel.FacebookGate {
	return &fbmodel.FacebookGate{
		ID:        "gate-1",
		Name:      "Test Page",
		MetaAppID: "app-1",
		PageID:    "page-1",
		PageName:  "Page Name",
		Status:    sharedmodel.StatusActive,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func newFacebookHandler(svc *mockFacebookService) *FacebookHandler {
	return NewFacebookHandler(noopLogger, svc, nil)
}

// -- tests --

func TestCreateFacebookGate_Success(t *testing.T) {
	svc := &mockFacebookService{
		createFn: func(_ context.Context, req fbmodel.CreateFacebook) (*fbmodel.FacebookGate, error) {
			if req.Name != "Test Page" {
				t.Errorf("unexpected name: %s", req.Name)
			}
			if req.Dc != 42 {
				t.Errorf("unexpected domain id: %d", req.Dc)
			}
			return stubGate(), nil
		},
	}

	h := newFacebookHandler(svc)
	resp, err := h.CreateFacebookGate(ctxWithAuth(42), &impb.ProviderCreateFacebookGateRequest{
		Name:      "Test Page",
		MetaAppId: "app-1",
		PageId:    "page-1",
		PageToken: "tok",
		Peer:      &impb.Peer{Sub: "sub", Iss: "iss"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Item.Id != "gate-1" {
		t.Errorf("unexpected gate id: %s", resp.Item.Id)
	}
}

func TestCreateFacebookGate_MissingAuth(t *testing.T) {
	h := newFacebookHandler(&mockFacebookService{})
	_, err := h.CreateFacebookGate(context.Background(), &impb.ProviderCreateFacebookGateRequest{})
	if err == nil {
		t.Fatal("expected unauthenticated error, got nil")
	}
}

func TestCreateFacebookGate_ServiceError(t *testing.T) {
	svc := &mockFacebookService{
		createFn: func(_ context.Context, _ fbmodel.CreateFacebook) (*fbmodel.FacebookGate, error) {
			return nil, errors.New("db error")
		},
	}
	h := newFacebookHandler(svc)
	_, err := h.CreateFacebookGate(ctxWithAuth(1), &impb.ProviderCreateFacebookGateRequest{
		Peer: &impb.Peer{},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetFacebookGate_Success(t *testing.T) {
	svc := &mockFacebookService{
		getFn: func(_ context.Context, id string) (*fbmodel.FacebookGate, error) {
			if id != "gate-1" {
				t.Errorf("unexpected id: %s", id)
			}
			return stubGate(), nil
		},
	}
	h := newFacebookHandler(svc)
	resp, err := h.GetFacebookGate(context.Background(), &impb.ProviderGetFacebookGateRequest{Id: "gate-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Item.Id != "gate-1" {
		t.Errorf("unexpected id: %s", resp.Item.Id)
	}
}

func TestGetFacebookGate_NotFound(t *testing.T) {
	svc := &mockFacebookService{
		getFn: func(_ context.Context, _ string) (*fbmodel.FacebookGate, error) {
			return nil, errors.New("not found")
		},
	}
	h := newFacebookHandler(svc)
	_, err := h.GetFacebookGate(context.Background(), &impb.ProviderGetFacebookGateRequest{Id: "missing"})
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
}

func TestUpdateFacebookGate_Success(t *testing.T) {
	svc := &mockFacebookService{
		updateFn: func(_ context.Context, req fbmodel.UpdateFacebook) (*fbmodel.FacebookGate, error) {
			if req.ID != "gate-1" {
				t.Errorf("unexpected id: %s", req.ID)
			}
			g := stubGate()
			g.Name = *req.Name
			return g, nil
		},
	}
	h := newFacebookHandler(svc)
	resp, err := h.UpdateFacebookGate(context.Background(), &impb.ProviderUpdateFacebookGateRequest{
		Id:   "gate-1",
		Name: "Updated Name",
		Peer: &impb.Peer{Sub: "sub", Iss: "iss"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Item.Name != "Updated Name" {
		t.Errorf("unexpected name: %s", resp.Item.Name)
	}
}

func TestUpdateFacebookGate_ServiceError(t *testing.T) {
	svc := &mockFacebookService{
		updateFn: func(_ context.Context, _ fbmodel.UpdateFacebook) (*fbmodel.FacebookGate, error) {
			return nil, errors.New("update failed")
		},
	}
	h := newFacebookHandler(svc)
	_, err := h.UpdateFacebookGate(context.Background(), &impb.ProviderUpdateFacebookGateRequest{
		Id:   "gate-1",
		Peer: &impb.Peer{},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteFacebookGate_Success(t *testing.T) {
	svc := &mockFacebookService{
		deleteFn: func(_ context.Context, id string) (*fbmodel.FacebookGate, error) {
			if id != "gate-1" {
				t.Errorf("unexpected id: %s", id)
			}
			return stubGate(), nil
		},
	}
	h := newFacebookHandler(svc)
	resp, err := h.DeleteFacebookGate(context.Background(), &impb.ProviderDeleteFacebookGateRequest{Id: "gate-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Item.Id != "gate-1" {
		t.Errorf("unexpected id: %s", resp.Item.Id)
	}
}

func TestDeleteFacebookGate_ServiceError(t *testing.T) {
	svc := &mockFacebookService{
		deleteFn: func(_ context.Context, _ string) (*fbmodel.FacebookGate, error) {
			return nil, errors.New("delete failed")
		},
	}
	h := newFacebookHandler(svc)
	_, err := h.DeleteFacebookGate(context.Background(), &impb.ProviderDeleteFacebookGateRequest{Id: "gate-1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGateToProto_NilGate(t *testing.T) {
	h := newFacebookHandler(&mockFacebookService{})
	if h.gateToProto(nil) != nil {
		t.Error("expected nil proto for nil gate")
	}
}

func TestGateToProto_FieldMapping(t *testing.T) {
	h := newFacebookHandler(&mockFacebookService{})
	g := stubGate()
	proto := h.gateToProto(g)

	if proto.Id != g.ID {
		t.Errorf("id mismatch: %s vs %s", proto.Id, g.ID)
	}
	if proto.Name != g.Name {
		t.Errorf("name mismatch: %s vs %s", proto.Name, g.Name)
	}
	if proto.MetaAppId != g.MetaAppID {
		t.Errorf("meta_app_id mismatch")
	}
	if proto.PageId != g.PageID {
		t.Errorf("page_id mismatch")
	}
	if proto.Enabled != g.Enabled {
		t.Errorf("enabled mismatch")
	}
}
