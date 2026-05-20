package service

import (
	"context"

	wamodel "github.com/webitel/im-providers-service/internal/whatsapp/model"
)

var _ WhatsAppManager = (*WhatsAppService)(nil)

type WhatsAppManager interface {
	CreateGate(ctx context.Context, req wamodel.CreateWhatsApp) (*wamodel.WhatsAppGate, error)
	GetGate(ctx context.Context, id string) (*wamodel.WhatsAppGate, error)
	UpdateGate(ctx context.Context, req wamodel.UpdateWhatsApp) (*wamodel.WhatsAppGate, error)
	DeleteGate(ctx context.Context, id string) (*wamodel.WhatsAppGate, error)
}

type WhatsAppService struct{}

func NewWhatsAppService() *WhatsAppService {
	return &WhatsAppService{}
}

// CreateGate implements [WhatsAppManager].
func (w *WhatsAppService) CreateGate(ctx context.Context, req wamodel.CreateWhatsApp) (*wamodel.WhatsAppGate, error) {
	panic("unimplemented")
}

// DeleteGate implements [WhatsAppManager].
func (w *WhatsAppService) DeleteGate(ctx context.Context, id string) (*wamodel.WhatsAppGate, error) {
	panic("unimplemented")
}

// GetGate implements [WhatsAppManager].
func (w *WhatsAppService) GetGate(ctx context.Context, id string) (*wamodel.WhatsAppGate, error) {
	panic("unimplemented")
}

// UpdateGate implements [WhatsAppManager].
func (w *WhatsAppService) UpdateGate(ctx context.Context, req wamodel.UpdateWhatsApp) (*wamodel.WhatsAppGate, error) {
	panic("unimplemented")
}
