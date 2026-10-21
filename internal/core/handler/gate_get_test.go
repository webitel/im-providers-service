package handler

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	"github.com/webitel/im-providers-service/infra/auth"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
)

type testIdentity struct{ domainID int64 }

func (i testIdentity) GetContactID() string { return "" }
func (i testIdentity) GetDomainID() int64   { return i.domainID }
func (i testIdentity) GetName() string      { return "" }

func withDomain(domainID int64) context.Context {
	return context.WithValue(context.Background(), auth.AuthContextKey, testIdentity{domainID: domainID})
}

func TestResolveDomainID(t *testing.T) {
	tests := []struct {
		name      string
		ctx       context.Context
		requested int64
		want      int64
		wantCode  codes.Code
	}{
		{"user identity wins over request", withDomain(7), 9, 7, codes.OK},
		{"service call uses request", context.Background(), 9, 9, codes.OK},
		{"identity without domain falls back to request", withDomain(0), 9, 9, codes.OK},
		{"no domain anywhere", context.Background(), 0, 0, codes.InvalidArgument},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveDomainID(tt.ctx, tt.requested)
			if code := status.Code(err); code != tt.wantCode {
				t.Fatalf("code = %v, want %v (err: %v)", code, tt.wantCode, err)
			}
			if got != tt.want {
				t.Errorf("domain = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestListGates_ScopesAndFilters(t *testing.T) {
	var got sharedmodel.ListFilter
	svc := &mockGateService{
		listFn: func(_ context.Context, f sharedmodel.ListFilter) ([]*sharedmodel.GateSummary, bool, error) {
			got = f
			return nil, false, nil
		},
	}
	h := NewGateHandler(noopLogger, svc, nil)

	_, err := h.ListGates(withDomain(3), &impb.ProviderListGatesRequest{
		DomainId: 99, // ignored: the caller's own domain wins
		Types:    []impb.ProviderType{impb.ProviderType_PROVIDER_TYPE_VIBER_BM, impb.ProviderType_PROVIDER_TYPE_META_APP},
		Status:   impb.ProviderStatus_PROVIDER_STATUS_INACTIVE,
		Q:        "shop",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.DomainID != 3 {
		t.Errorf("DomainID = %d, want 3", got.DomainID)
	}
	if len(got.Types) != 1 || got.Types[0] != sharedmodel.TypeViberBM {
		t.Errorf("Types = %v, want [viber_bm]", got.Types)
	}
	if got.Status != sharedmodel.StatusDisabled {
		t.Errorf("Status = %v, want disabled", got.Status)
	}
	if got.Q != "shop" {
		t.Errorf("Q = %q, want shop", got.Q)
	}
}

func TestListGates_MissingDomain(t *testing.T) {
	h := NewGateHandler(noopLogger, &mockGateService{}, nil)

	_, err := h.ListGates(context.Background(), &impb.ProviderListGatesRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %v, want InvalidArgument", status.Code(err))
	}
}

func TestGetGate(t *testing.T) {
	gate := stubGateSummary("g1", sharedmodel.TypeWhatsApp)
	gate.Address = "+380671234567"

	tests := []struct {
		name     string
		ctx      context.Context
		req      *impb.GetGateRequest
		getErr   error
		wantCode codes.Code
	}{
		{"found", context.Background(), &impb.GetGateRequest{Id: "g1", DomainId: 5}, nil, codes.OK},
		{"not found", context.Background(), &impb.GetGateRequest{Id: "g1", DomainId: 5}, sharedstore.ErrNotFound, codes.NotFound},
		{"store failure", context.Background(), &impb.GetGateRequest{Id: "g1", DomainId: 5}, errors.New("db down"), codes.Internal},
		{"missing id", context.Background(), &impb.GetGateRequest{DomainId: 5}, nil, codes.InvalidArgument},
		{"missing domain", context.Background(), &impb.GetGateRequest{Id: "g1"}, nil, codes.InvalidArgument},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockGateService{
				getFn: func(_ context.Context, domainID int64, id string) (*sharedmodel.GateSummary, error) {
					if domainID != 5 || id != "g1" {
						t.Errorf("GetGate(%d, %q), want (5, g1)", domainID, id)
					}
					if tt.getErr != nil {
						return nil, tt.getErr
					}
					return gate, nil
				},
			}
			h := NewGateHandler(noopLogger, svc, nil)

			resp, err := h.GetGate(tt.ctx, tt.req)
			if code := status.Code(err); code != tt.wantCode {
				t.Fatalf("code = %v, want %v (err: %v)", code, tt.wantCode, err)
			}
			if tt.wantCode != codes.OK {
				return
			}

			item := resp.GetItem()
			if item.GetId() != "g1" || item.GetAddress() != "+380671234567" || item.GetType() != impb.ProviderType_PROVIDER_TYPE_WHATSAPP {
				t.Errorf("unexpected item: %+v", item)
			}
		})
	}
}
