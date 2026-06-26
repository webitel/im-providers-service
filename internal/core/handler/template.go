package handler

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	"github.com/webitel/im-providers-service/infra/auth"
	corestore "github.com/webitel/im-providers-service/internal/core/store"
)

type GateTemplateHandler struct {
	logger *slog.Logger
	store  corestore.TemplateStore
	impb.UnimplementedGateTemplateServiceServer
}

func NewGateTemplateHandler(logger *slog.Logger, store corestore.TemplateStore) *GateTemplateHandler {
	return &GateTemplateHandler{
		logger: logger.With("handler", "gate_template"),
		store:  store,
	}
}

func (h *GateTemplateHandler) SetGateTemplate(ctx context.Context, req *impb.ProviderSetGateTemplateRequest) (*impb.ProviderSetGateTemplateResponse, error) {
	identity, ok := auth.GetIdentityFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing identity in context")
	}

	if err := h.store.SetTemplate(ctx, req.GetGateId(), req.GetEventType(), req.GetTemplate(), identity.GetDomainID()); err != nil {
		h.logger.ErrorContext(ctx, "set gate template failed", "gate_id", req.GetGateId(), "event_type", req.GetEventType(), "err", err)
		return nil, status.Error(codes.Internal, "failed to set template")
	}

	return &impb.ProviderSetGateTemplateResponse{
		Item: &impb.ProviderGateTemplate{
			GateId:    req.GetGateId(),
			EventType: req.GetEventType(),
			Template:  req.GetTemplate(),
		},
	}, nil
}

func (h *GateTemplateHandler) DeleteGateTemplate(ctx context.Context, req *impb.ProviderDeleteGateTemplateRequest) (*impb.ProviderDeleteGateTemplateResponse, error) {
	err := h.store.DeleteTemplate(ctx, req.GetGateId(), req.GetEventType())
	if errors.Is(err, corestore.ErrNotFound) {
		return nil, status.Errorf(codes.NotFound, "template not found for event type %q", req.GetEventType())
	}
	if err != nil {
		h.logger.ErrorContext(ctx, "delete gate template failed", "gate_id", req.GetGateId(), "event_type", req.GetEventType(), "err", err)
		return nil, status.Error(codes.Internal, "failed to delete template")
	}

	return &impb.ProviderDeleteGateTemplateResponse{}, nil
}

func (h *GateTemplateHandler) ListGateTemplates(ctx context.Context, req *impb.ProviderListGateTemplatesRequest) (*impb.ProviderListGateTemplatesResponse, error) {
	rows, err := h.store.ListTemplates(ctx, req.GetGateId())
	if err != nil {
		h.logger.ErrorContext(ctx, "list gate templates failed", "gate_id", req.GetGateId(), "err", err)
		return nil, status.Error(codes.Internal, "failed to list templates")
	}

	items := make([]*impb.ProviderGateTemplate, len(rows))
	for i, r := range rows {
		items[i] = &impb.ProviderGateTemplate{
			GateId:    r.GateID,
			EventType: r.EventType,
			Template:  r.Template,
		}
	}

	return &impb.ProviderListGateTemplatesResponse{Items: items}, nil
}
