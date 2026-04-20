package service

import (
	"context"

	"github.com/webitel/im-providers-service/internal/domain/model"
)

var _ WhatsAppManager = (*WhatsAppService)(nil)

type WhatsAppManager interface {
	CreateGate(ctx context.Context, req model.CreateWhatsApp) (*model.WhatsAppGate, error)
	GetGate(ctx context.Context, id string) (*model.WhatsAppGate, error)
	UpdateGate(ctx context.Context, req model.UpdateWhatsApp) (*model.WhatsAppGate, error)
	DeleteGate(ctx context.Context, id string) (*model.WhatsAppGate, error)
}

type WhatsAppService struct{}

func NewWhatsAppService() *WhatsAppService {
	return &WhatsAppService{}
}

// CreateGate implements [WhatsAppManager].
func (w *WhatsAppService) CreateGate(ctx context.Context, req model.CreateWhatsApp) (*model.WhatsAppGate, error) {
	panic("unimplemented")
}

// DeleteGate implements [WhatsAppManager].
func (w *WhatsAppService) DeleteGate(ctx context.Context, id string) (*model.WhatsAppGate, error) {
	panic("unimplemented")
}

// GetGate implements [WhatsAppManager].
func (w *WhatsAppService) GetGate(ctx context.Context, id string) (*model.WhatsAppGate, error) {
	panic("unimplemented")
}

// UpdateGate implements [WhatsAppManager].
func (w *WhatsAppService) UpdateGate(ctx context.Context, req model.UpdateWhatsApp) (*model.WhatsAppGate, error) {
	panic("unimplemented")
}
