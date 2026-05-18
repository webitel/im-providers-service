package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	"github.com/webitel/im-providers-service/internal/domain/model"
)

type mockGateService struct {
	listFn func(ctx context.Context, f model.ListFilter) ([]*model.GateSummary, bool, error)
}

func (m *mockGateService) ListGates(ctx context.Context, f model.ListFilter) ([]*model.GateSummary, bool, error) {
	return m.listFn(ctx, f)
}

func stubGateSummary(id string, t model.GateType) *model.GateSummary {
	appID := "app-1"
	return &model.GateSummary{
		ID:            id,
		Name:          "Gate " + id,
		Type:          t,
		Status:        model.StatusActive,
		WebhookURL:    "https://example.com/wh",
		Contact:       "contact",
		ProviderAppID: &appID,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func TestListGates_Success(t *testing.T) {
	svc := &mockGateService{
		listFn: func(_ context.Context, f model.ListFilter) ([]*model.GateSummary, bool, error) {
			if f.Size != 10 {
				t.Errorf("unexpected size: %d", f.Size)
			}
			return []*model.GateSummary{
				stubGateSummary("g1", model.TypeFacebook),
				stubGateSummary("g2", model.TypeWhatsApp),
			}, true, nil
		},
	}
	h := NewGateHandler(noopLogger, svc)
	resp, err := h.ListGates(context.Background(), &impb.ProviderListGatesRequest{
		Page: 1,
		Size: 10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(resp.Items))
	}
	if !resp.Next {
		t.Error("expected next=true")
	}
}

func TestListGates_DefaultSize(t *testing.T) {
	svc := &mockGateService{
		listFn: func(_ context.Context, f model.ListFilter) ([]*model.GateSummary, bool, error) {
			if f.Size != 20 {
				t.Errorf("expected default size 20, got %d", f.Size)
			}
			return nil, false, nil
		},
	}
	h := NewGateHandler(noopLogger, svc)
	resp, err := h.ListGates(context.Background(), &impb.ProviderListGatesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Items) != 0 {
		t.Errorf("expected 0 items")
	}
}

func TestListGates_ServiceError(t *testing.T) {
	svc := &mockGateService{
		listFn: func(_ context.Context, _ model.ListFilter) ([]*model.GateSummary, bool, error) {
			return nil, false, errors.New("db error")
		},
	}
	h := NewGateHandler(noopLogger, svc)
	_, err := h.ListGates(context.Background(), &impb.ProviderListGatesRequest{Size: 10})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListGates_NilProviderAppID(t *testing.T) {
	svc := &mockGateService{
		listFn: func(_ context.Context, _ model.ListFilter) ([]*model.GateSummary, bool, error) {
			return []*model.GateSummary{{
				ID:            "g1",
				Name:          "Gate",
				Type:          model.TypeFacebook,
				Status:        model.StatusActive,
				ProviderAppID: nil,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}}, false, nil
		},
	}
	h := NewGateHandler(noopLogger, svc)
	resp, err := h.ListGates(context.Background(), &impb.ProviderListGatesRequest{Size: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Items[0].ProviderAppId != "" {
		t.Errorf("expected empty provider app id, got %s", resp.Items[0].ProviderAppId)
	}
}

func TestToProtoType(t *testing.T) {
	cases := []struct {
		in   model.GateType
		want impb.ProviderType
	}{
		{model.TypeFacebook, impb.ProviderType_PROVIDER_TYPE_FACEBOOK},
		{model.TypeInstagram, impb.ProviderType_PROVIDER_TYPE_INSTAGRAM},
		{model.TypeWhatsApp, impb.ProviderType_PROVIDER_TYPE_WHATSAPP},
		{model.TypeTelegramBot, impb.ProviderType_PROVIDER_TYPE_TELEGRAM_BOT},
		{model.TypeTelegramApp, impb.ProviderType_PROVIDER_TYPE_TELEGRAM_APP},
		{model.GateType(99), impb.ProviderType_PROVIDER_TYPE_UNSPECIFIED},
	}
	for _, c := range cases {
		got := toProtoType(c.in)
		if got != c.want {
			t.Errorf("toProtoType(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestToProtoStatus(t *testing.T) {
	cases := []struct {
		in   model.GateStatus
		want impb.ProviderStatus
	}{
		{model.StatusActive, impb.ProviderStatus_PROVIDER_STATUS_ACTIVE},
		{model.StatusDisabled, impb.ProviderStatus_PROVIDER_STATUS_INACTIVE},
		{model.StatusError, impb.ProviderStatus_PROVIDER_STATUS_ERROR},
		{model.GateStatus(99), impb.ProviderStatus_PROVIDER_STATUS_UNSPECIFIED},
	}
	for _, c := range cases {
		got := toProtoStatus(c.in)
		if got != c.want {
			t.Errorf("toProtoStatus(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}
