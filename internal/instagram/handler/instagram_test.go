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
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	igmodel "github.com/webitel/im-providers-service/internal/instagram/model"
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

type mockInstagramService struct {
	createFn func(ctx context.Context, req igmodel.CreateInstagram) (*igmodel.InstagramGate, error)
	getFn    func(ctx context.Context, id string, dc int64) (*igmodel.InstagramGate, error)
	updateFn func(ctx context.Context, req igmodel.UpdateInstagram) (*igmodel.InstagramGate, error)
	deleteFn func(ctx context.Context, id string, dc int64) (*igmodel.InstagramGate, error)
}

func (m *mockInstagramService) CreateGate(ctx context.Context, req igmodel.CreateInstagram) (*igmodel.InstagramGate, error) {
	return m.createFn(ctx, req)
}

func (m *mockInstagramService) GetGate(ctx context.Context, id string, dc int64) (*igmodel.InstagramGate, error) {
	return m.getFn(ctx, id, dc)
}

func (m *mockInstagramService) UpdateGate(ctx context.Context, req igmodel.UpdateInstagram) (*igmodel.InstagramGate, error) {
	return m.updateFn(ctx, req)
}

func (m *mockInstagramService) DeleteGate(ctx context.Context, id string, dc int64) (*igmodel.InstagramGate, error) {
	return m.deleteFn(ctx, id, dc)
}

// -- helpers --

func stubInstagramGate() *igmodel.InstagramGate {
	return &igmodel.InstagramGate{
		ID:                "gate-1",
		Name:              "Test Business Account",
		MetaAppID:         "app-1",
		BusinessAccountID: "biz-acct-1",
		Status:            sharedmodel.StatusActive,
		Enabled:           true,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

func instagramHandler(svc *mockInstagramService) *InstagramHandler {
	return NewInstagramHandler(noopLogger, svc, nil)
}

// -- tests --

func TestCreateInstagramGate_Success(t *testing.T) {
	svc := &mockInstagramService{
		createFn: func(_ context.Context, req igmodel.CreateInstagram) (*igmodel.InstagramGate, error) {
			if req.Name != "Test Business Account" {
				t.Errorf("unexpected name: %s", req.Name)
			}

			if req.Dc != 42 {
				t.Errorf("unexpected domain id: %d", req.Dc)
			}

			return stubInstagramGate(), nil
		},
	}

	h := instagramHandler(svc)

	resp, err := h.CreateInstagramGate(ctxWithAuth(42), &impb.ProviderCreateInstagramGateRequest{
		Name:                 "Test Business Account",
		MetaAppId:            "app-1",
		BusinessAccountId:    "biz-acct-1",
		BusinessAccountToken: "tok",
		Peer:                 &impb.Peer{Sub: "sub", Iss: "iss"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.GetItem().GetId() != "gate-1" {
		t.Errorf("unexpected gate id: %s", resp.GetItem().GetId())
	}
}

func TestCreateInstagramGate_MissingAuth(t *testing.T) {
	h := instagramHandler(&mockInstagramService{})

	_, err := h.CreateInstagramGate(context.Background(), &impb.ProviderCreateInstagramGateRequest{})
	if err == nil {
		t.Fatal("expected unauthenticated error, got nil")
	}
}

func TestCreateInstagramGate_ServiceError(t *testing.T) {
	svc := &mockInstagramService{
		createFn: func(_ context.Context, _ igmodel.CreateInstagram) (*igmodel.InstagramGate, error) {
			return nil, errors.New("db error")
		},
	}

	h := instagramHandler(svc)

	_, err := h.CreateInstagramGate(ctxWithAuth(1), &impb.ProviderCreateInstagramGateRequest{
		Peer: &impb.Peer{},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetInstagramGate_Success(t *testing.T) {
	svc := &mockInstagramService{
		getFn: func(_ context.Context, id string, _ int64) (*igmodel.InstagramGate, error) {
			if id != "gate-1" {
				t.Errorf("unexpected id: %s", id)
			}

			return stubInstagramGate(), nil
		},
	}

	h := instagramHandler(svc)

	resp, err := h.GetInstagramGate(ctxWithAuth(1), &impb.ProviderGetInstagramGateRequest{Id: "gate-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.GetItem().GetId() != "gate-1" {
		t.Errorf("unexpected id: %s", resp.GetItem().GetId())
	}
}

func TestGetInstagramGate_NotFound(t *testing.T) {
	svc := &mockInstagramService{
		getFn: func(_ context.Context, _ string, _ int64) (*igmodel.InstagramGate, error) {
			return nil, errors.New("not found")
		},
	}

	h := instagramHandler(svc)

	_, err := h.GetInstagramGate(ctxWithAuth(1), &impb.ProviderGetInstagramGateRequest{Id: "missing"})
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
}

func TestUpdateInstagramGate_Success(t *testing.T) {
	svc := &mockInstagramService{
		updateFn: func(_ context.Context, req igmodel.UpdateInstagram) (*igmodel.InstagramGate, error) {
			if req.ID != "gate-1" {
				t.Errorf("unexpected id: %s", req.ID)
			}

			g := stubInstagramGate()
			g.Name = *req.Name

			return g, nil
		},
	}

	h := instagramHandler(svc)

	_, err := h.UpdateInstagramGate(ctxWithAuth(1), &impb.ProviderUpdateInstagramGateRequest{
		Id:   "gate-1",
		Name: "Updated Name",
		Peer: &impb.Peer{Sub: "sub", Iss: "iss"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateInstagramGate_ServiceError(t *testing.T) {
	svc := &mockInstagramService{
		updateFn: func(_ context.Context, _ igmodel.UpdateInstagram) (*igmodel.InstagramGate, error) {
			return nil, errors.New("update failed")
		},
	}

	h := instagramHandler(svc)

	_, err := h.UpdateInstagramGate(ctxWithAuth(1), &impb.ProviderUpdateInstagramGateRequest{
		Id:   "gate-1",
		Peer: &impb.Peer{},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteInstagramGate_Success(t *testing.T) {
	svc := &mockInstagramService{
		deleteFn: func(_ context.Context, id string, _ int64) (*igmodel.InstagramGate, error) {
			if id != "gate-1" {
				t.Errorf("unexpected id: %s", id)
			}

			return stubInstagramGate(), nil
		},
	}

	h := instagramHandler(svc)

	resp, err := h.DeleteInstagramGate(ctxWithAuth(1), &impb.ProviderDeleteInstagramGateRequest{Id: "gate-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.GetItem().GetId() != "gate-1" {
		t.Errorf("unexpected id: %s", resp.GetItem().GetId())
	}
}

func TestDeleteInstagramGate_ServiceError(t *testing.T) {
	svc := &mockInstagramService{
		deleteFn: func(_ context.Context, _ string, _ int64) (*igmodel.InstagramGate, error) {
			return nil, errors.New("delete failed")
		},
	}

	h := instagramHandler(svc)

	_, err := h.DeleteInstagramGate(ctxWithAuth(1), &impb.ProviderDeleteInstagramGateRequest{Id: "gate-1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGateToProto_NilGate(t *testing.T) {
	h := instagramHandler(&mockInstagramService{})
	if h.gateToProto(nil) != nil {
		t.Error("expected nil proto for nil gate")
	}
}

func TestGateToProto_FieldMapping(t *testing.T) {
	h := instagramHandler(&mockInstagramService{})
	g := stubInstagramGate()
	proto := h.gateToProto(g)

	if proto.GetId() != g.ID {
		t.Errorf("id mismatch: %s vs %s", proto.GetId(), g.ID)
	}

	if proto.GetName() != g.Name {
		t.Errorf("name mismatch: %s vs %s", proto.GetName(), g.Name)
	}

	if proto.GetMetaAppId() != g.MetaAppID {
		t.Errorf("meta_app_id mismatch")
	}

	if proto.GetBusinessAccountId() != g.BusinessAccountID {
		t.Errorf("business_account_id mismatch")
	}

	if proto.GetEnabled() != g.Enabled {
		t.Errorf("enabled mismatch")
	}
}
