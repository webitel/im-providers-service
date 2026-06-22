package handler

import (
	"context"
	"log/slog"

	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	fbmodel "github.com/webitel/im-providers-service/internal/facebook/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (f *FacebookHandler) SetPersistentMenu(ctx context.Context, req *impb.ProviderSetPersistentMenuRequest) (*impb.ProviderSetPersistentMenuResponse, error) {
	log := f.logger.With(slog.String("method", "SetPersistentMenu"), slog.String("gate_id", req.GetGateId()))
	if req.GetGateId() == "" {
		return nil, status.Error(codes.InvalidArgument, "gate_id is required")
	}

	items := protoMenuItemsToModel(req.GetItems())
	log.InfoContext(ctx, "setting persistent menu", slog.Int("items", len(items)), slog.Bool("composer_disabled", req.GetComposerInputDisabled()))

	if err := f.srv.SetPersistentMenu(ctx, req.GetGateId(), items, req.GetComposerInputDisabled()); err != nil {
		log.ErrorContext(ctx, "failed to set persistent menu", slog.String("error", err.Error()))
		return nil, toStatus(err, "set persistent menu")
	}

	log.InfoContext(ctx, "persistent menu set successfully")
	return &impb.ProviderSetPersistentMenuResponse{}, nil
}

func (f *FacebookHandler) DeletePersistentMenu(ctx context.Context, req *impb.ProviderDeletePersistentMenuRequest) (*impb.ProviderDeletePersistentMenuResponse, error) {
	log := f.logger.With(slog.String("method", "DeletePersistentMenu"), slog.String("gate_id", req.GetGateId()))
	if req.GetGateId() == "" {
		return nil, status.Error(codes.InvalidArgument, "gate_id is required")
	}

	log.InfoContext(ctx, "deleting persistent menu")
	if err := f.srv.DeletePersistentMenu(ctx, req.GetGateId()); err != nil {
		log.ErrorContext(ctx, "failed to delete persistent menu", slog.String("error", err.Error()))
		return nil, toStatus(err, "delete persistent menu")
	}

	log.InfoContext(ctx, "persistent menu deleted")
	return &impb.ProviderDeletePersistentMenuResponse{}, nil
}

func (f *FacebookHandler) SetGetStarted(ctx context.Context, req *impb.ProviderSetGetStartedRequest) (*impb.ProviderSetGetStartedResponse, error) {
	log := f.logger.With(slog.String("method", "SetGetStarted"), slog.String("gate_id", req.GetGateId()))
	if req.GetGateId() == "" {
		return nil, status.Error(codes.InvalidArgument, "gate_id is required")
	}
	if req.GetPayload() == "" {
		return nil, status.Error(codes.InvalidArgument, "payload is required")
	}

	log.InfoContext(ctx, "setting get started button", slog.String("payload", req.GetPayload()))
	if err := f.srv.SetGetStarted(ctx, req.GetGateId(), req.GetPayload()); err != nil {
		log.ErrorContext(ctx, "failed to set get started button", slog.String("error", err.Error()))
		return nil, toStatus(err, "set get started")
	}

	log.InfoContext(ctx, "get started button set successfully")
	return &impb.ProviderSetGetStartedResponse{}, nil
}

func (f *FacebookHandler) DeleteGetStarted(ctx context.Context, req *impb.ProviderDeleteGetStartedRequest) (*impb.ProviderDeleteGetStartedResponse, error) {
	log := f.logger.With(slog.String("method", "DeleteGetStarted"), slog.String("gate_id", req.GetGateId()))
	if req.GetGateId() == "" {
		return nil, status.Error(codes.InvalidArgument, "gate_id is required")
	}

	log.InfoContext(ctx, "deleting get started button")
	if err := f.srv.DeleteGetStarted(ctx, req.GetGateId()); err != nil {
		log.ErrorContext(ctx, "failed to delete get started button", slog.String("error", err.Error()))
		return nil, toStatus(err, "delete get started")
	}

	log.InfoContext(ctx, "get started button deleted")
	return &impb.ProviderDeleteGetStartedResponse{}, nil
}

// protoMenuItemsToModel maps proto ProviderMenuItem repeated field to domain model recursively.
func protoMenuItemsToModel(items []*impb.ProviderMenuItem) []fbmodel.MenuItem {
	result := make([]fbmodel.MenuItem, 0, len(items))
	for _, item := range items {
		m := fbmodel.MenuItem{Title: item.GetTitle()}
		switch a := item.GetAction().(type) {
		case *impb.ProviderMenuItem_Payload:
			m.Payload = a.Payload
		case *impb.ProviderMenuItem_Url:
			m.URL = a.Url
		case *impb.ProviderMenuItem_Nested:
			if a.Nested != nil {
				m.Nested = protoMenuItemsToModel(a.Nested.GetItems())
			}
		}
		result = append(result, m)
	}
	return result
}
