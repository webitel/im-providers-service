package handler

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	coremodel "github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/im-providers-service/internal/telegram/bot/model"
)

var _ impb.TelegramBotServiceServer = &TelegramBotHandler{}

type TelegramBotService interface {
	CreateTelegramBot(context.Context, *model.CreateGate) (*model.Gate, error)
	DeleteTelegramBot(context.Context, uuid.UUID) (*model.Gate, error)
	GetTelegramBot(context.Context, uuid.UUID) (*model.Gate, error)
	UpdateTelegramBot(context.Context, *model.UpdateGate) (*model.Gate, error)
}

func NewTelegramBotHandler(service TelegramBotService, log *slog.Logger) *TelegramBotHandler {
	return &TelegramBotHandler{
		service: service,
		log:     log,
	}
}

type TelegramBotHandler struct {
	impb.UnimplementedTelegramBotServiceServer

	service TelegramBotService
	log     *slog.Logger
}

// CreateTelegramBotGate implements [provider.TelegramBotServiceServer].
func (t *TelegramBotHandler) CreateTelegramBotGate(ctx context.Context, req *impb.CreateTelegramBotGateRequest) (*impb.CreateTelegramBotGateResponse, error) {
	var peer *coremodel.Peer

	if req.GetPeer() != nil {
		peer = &coremodel.Peer{Sub: req.GetPeer().Sub, Iss: req.GetPeer().Iss}
	}
	gate, err := t.service.CreateTelegramBot(ctx, &model.CreateGate{
		Name:          req.GetName(),
		Token:         req.GetApiToken(),
		Bot:           peer,
		Enabled:       req.GetEnabled(),
		URI:           req.GetUri(),
		WebhookSecret: req.GetWebhookSecret(),
	})
	if err != nil {
		return nil, err
	}
	return &impb.CreateTelegramBotGateResponse{
		Item: parseGateToTelegramBotGate(gate),
	}, nil
}

func parseGateToTelegramBotGate(gate *model.Gate) *impb.ProviderTelegramBotGate {
	return &impb.ProviderTelegramBotGate{
		Id:        gate.ID.String(),
		Name:      gate.Name,
		Status:    ParseGateStatus(gate.Enabled),
		CreatedAt: gate.CreatedAt,
		UpdatedAt: gate.UpdatedAt,
	}
}

func ParseGateStatus(status bool) impb.ProviderStatus {
	if status {
		return impb.ProviderStatus_PROVIDER_STATUS_ACTIVE
	}
	return impb.ProviderStatus_PROVIDER_STATUS_INACTIVE
}

// DeleteTelegramBotGate implements [provider.TelegramBotServiceServer].
func (t *TelegramBotHandler) DeleteTelegramBotGate(ctx context.Context, req *impb.DeleteTelegramBotGateRequest) (*impb.DeleteTelegramBotGateResponse, error) {

	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, err
	}

	gate, err := t.service.DeleteTelegramBot(ctx, id)
	if err != nil {
		return nil, err
	}
	return &impb.DeleteTelegramBotGateResponse{
		Item: parseGateToTelegramBotGate(gate),
	}, nil
}

// GetTelegramBotGate implements [provider.TelegramBotServiceServer].
func (t *TelegramBotHandler) GetTelegramBotGate(ctx context.Context, req *impb.GetTelegramBotGateRequest) (*impb.GetTelegramBotGateResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, err
	}
	gate, err := t.service.GetTelegramBot(ctx, id)
	if err != nil {
		return nil, err
	}
	return &impb.GetTelegramBotGateResponse{
		Item: parseGateToTelegramBotGate(gate),
	}, nil
}

// UpdateTelegramBotGate implements [provider.TelegramBotServiceServer].
func (t *TelegramBotHandler) UpdateTelegramBotGate(ctx context.Context, req *impb.UpdateTelegramBotGateRequest) (*impb.UpdateTelegramBotGateResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, err
	}
	var peer coremodel.Peer
	if req.Peer != nil {
		peer = coremodel.Peer{
			Sub: req.Peer.Sub,
			Iss: req.Peer.Iss,
		}
	}
	internalReq := &model.UpdateGate{
		ID:            id,
		Name:          req.Name,
		Token:         req.ApiToken,
		Bot:           &peer,
		Enabled:       req.Enabled,
		WebhookSecret: req.WebhookSecret,
	}

	gate, err := t.service.UpdateTelegramBot(ctx, internalReq)
	if err != nil {
		return nil, err
	}
	return &impb.UpdateTelegramBotGateResponse{
		Item: parseGateToTelegramBotGate(gate),
	}, nil
}
