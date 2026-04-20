package grpc

import (
	"context"
	"log/slog"

	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
)

var _ impb.WhatsAppServiceServer = (*WhatsappHandler)(nil)

type WhatsappHandler struct {
	logger *slog.Logger
	impb.UnimplementedWhatsAppServiceServer
}

func NewWhatsAppHandler(logger *slog.Logger) *WhatsappHandler {
	return &WhatsappHandler{logger: logger}
}

// CreateWhatsAppGate implements [provider.WhatsAppServiceServer].
func (w *WhatsappHandler) CreateWhatsAppGate(context.Context, *impb.ProviderCreateWhatsAppGateRequest) (*impb.ProviderCreateWhatsAppGateResponse, error) {
	panic("unimplemented")
}

// DeleteWhatsAppGate implements [provider.WhatsAppServiceServer].
func (w *WhatsappHandler) DeleteWhatsAppGate(context.Context, *impb.ProviderDeleteWhatsAppGateRequest) (*impb.ProviderDeleteWhatsAppGateResponse, error) {
	panic("unimplemented")
}

// GetWhatsAppGate implements [provider.WhatsAppServiceServer].
func (w *WhatsappHandler) GetWhatsAppGate(context.Context, *impb.ProviderGetWhatsAppGateRequest) (*impb.ProviderGetWhatsAppGateResponse, error) {
	panic("unimplemented")
}

// UpdateWhatsAppGate implements [provider.WhatsAppServiceServer].
func (w *WhatsappHandler) UpdateWhatsAppGate(context.Context, *impb.ProviderUpdateWhatsAppGateRequest) (*impb.ProviderUpdateWhatsAppGateResponse, error) {
	panic("unimplemented")
}
