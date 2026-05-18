package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/webitel/im-providers-service/internal/domain/model"
)

type mockGateStore struct {
	listFn   func(ctx context.Context, f model.ListFilter) ([]*model.GateSummary, bool, error)
	deleteFn func(ctx context.Context, id string) error
}

func (m *mockGateStore) List(ctx context.Context, f model.ListFilter) ([]*model.GateSummary, bool, error) {
	return m.listFn(ctx, f)
}
func (m *mockGateStore) Delete(ctx context.Context, id string) error {
	return m.deleteFn(ctx, id)
}

func stubSummary(id string) *model.GateSummary {
	return &model.GateSummary{
		ID:        id,
		Name:      "Gate " + id,
		Type:      model.TypeFacebook,
		Status:    model.StatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestGateService_ListGates_Success(t *testing.T) {
	repo := &mockGateStore{
		listFn: func(_ context.Context, f model.ListFilter) ([]*model.GateSummary, bool, error) {
			if f.Page != 1 || f.Size != 5 {
				t.Errorf("unexpected filter: page=%d size=%d", f.Page, f.Size)
			}
			return []*model.GateSummary{stubSummary("g1"), stubSummary("g2")}, true, nil
		},
	}
	svc := NewGateService(repo, noopLogger)
	gates, next, err := svc.ListGates(context.Background(), model.ListFilter{Page: 1, Size: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gates) != 2 {
		t.Errorf("expected 2 gates, got %d", len(gates))
	}
	if !next {
		t.Error("expected next=true")
	}
}

func TestGateService_ListGates_Empty(t *testing.T) {
	repo := &mockGateStore{
		listFn: func(_ context.Context, _ model.ListFilter) ([]*model.GateSummary, bool, error) {
			return nil, false, nil
		},
	}
	svc := NewGateService(repo, noopLogger)
	gates, next, err := svc.ListGates(context.Background(), model.ListFilter{Page: 1, Size: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gates) != 0 {
		t.Errorf("expected 0 gates, got %d", len(gates))
	}
	if next {
		t.Error("expected next=false")
	}
}

func TestGateService_ListGates_StoreError(t *testing.T) {
	repo := &mockGateStore{
		listFn: func(_ context.Context, _ model.ListFilter) ([]*model.GateSummary, bool, error) {
			return nil, false, errors.New("db error")
		},
	}
	svc := NewGateService(repo, noopLogger)
	_, _, err := svc.ListGates(context.Background(), model.ListFilter{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGateService_ListGates_PassesFilter(t *testing.T) {
	repo := &mockGateStore{
		listFn: func(_ context.Context, f model.ListFilter) ([]*model.GateSummary, bool, error) {
			if f.Page != 3 || f.Size != 50 || f.Q != "search" {
				t.Errorf("filter mismatch: got %+v", f)
			}
			return nil, false, nil
		},
	}
	svc := NewGateService(repo, noopLogger)
	_, _, _ = svc.ListGates(context.Background(), model.ListFilter{Page: 3, Size: 50, Q: "search"})
}
