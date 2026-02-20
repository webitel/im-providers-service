package amqp

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
	"github.com/webitel/im-providers-service/internal/adapter/pubsub"
	"github.com/webitel/im-providers-service/internal/provider"
	"github.com/webitel/im-providers-service/internal/service/dto"
	"go.uber.org/fx"
)

const (
	// Base name for outbound message processing
	OutboundProcessorQueue = "im-providers.outbound-processor.v1"
	// Exchange name
	OutboundExchange = "im_message.events"
)

// [INJECTION_PARAMS] Structure for clean dependency injection with FX
type MessageHandlerParams struct {
	fx.In

	Logger *slog.Logger
	// [GROUP_COLLECTION] Collects all senders registered with group:"providers"
	Senders []provider.Sender `group:"providers"`
}

type MessageHandler struct {
	logger    *slog.Logger
	providers map[string]provider.Sender
}

// [CONSTRUCTOR] Updated to use MessageHandlerParams
func NewMessageHandler(p MessageHandlerParams) *MessageHandler {
	m := make(map[string]provider.Sender)
	for _, s := range p.Senders {
		m[s.Type()] = s
	}
	return &MessageHandler{
		logger:    p.Logger,
		providers: m,
	}
}

// RegisterHandlers routes AMQP topics to specific Go methods with unique queues.
func (h *MessageHandler) RegisterHandlers(router *message.Router, subProvider *pubsub.SubscriberProvider) error {
	configs := []struct {
		name    string
		topic   string
		handler message.NoPublishHandlerFunc
	}{
		{
			name:    "SEND_TEXT_V1",
			topic:   "im_provider.outbound.*.send.text.v1",
			handler: Bind(h, h.OnSendText),
		},
		{
			name:    "SEND_IMAGE_V1",
			topic:   "im_provider.outbound.*.send.image.v1",
			handler: Bind(h, h.OnSendImage),
		},
		{
			// [NEW] Added document handler configuration
			name:    "SEND_DOCUMENT_V1",
			topic:   "im_provider.outbound.*.send.document.v1",
			handler: Bind(h, h.OnSendDocument),
		},
	}

	for _, c := range configs {
		// [UNIQUE_HANDLER_QUEUE]
		// Generating a unique queue for each handler on this specific node instance.
		instanceID := uuid.NewString()[:8]
		handlerQueue := fmt.Sprintf("%s.%s.%s", OutboundProcessorQueue, instanceID, c.name)

		// Build subscriber using the injected provider
		sub, err := subProvider.Build(handlerQueue, OutboundExchange, c.topic)
		if err != nil {
			return fmt.Errorf("failed to build subscriber for %s: %w", c.name, err)
		}

		// Connect to the router with default middlewares
		router.AddConsumerHandler(
			c.name,
			c.topic,
			sub,
			c.handler,
		).AddMiddleware(
			middleware.Recoverer,
			middleware.Retry{MaxRetries: 3, InitialInterval: time.Second * 2}.Middleware,
			middleware.Timeout(time.Second*30),
		)
	}

	return nil
}

// --- [DOMAIN_HANDLERS] ---

func (h *MessageHandler) OnSendText(msg *message.Message, pType string, req *dto.MessageCreatedV1) error {
	p, ok := h.providers[pType]
	if !ok {
		h.logger.Warn("SKIPPED: provider_not_found", "type", pType)
		return nil
	}
	_, err := p.SendText(msg.Context(), req.ToDomain())
	return err
}

func (h *MessageHandler) OnSendImage(msg *message.Message, pType string, req *dto.MessageCreatedV1) error {
	p, ok := h.providers[pType]
	if !ok {
		h.logger.Warn("SKIPPED: provider_not_found", "type", pType)
		return nil
	}
	_, err := p.SendImage(msg.Context(), req.ToDomain())
	return err
}

// [NEW] OnSendDocument handles file/document outbound delivery
func (h *MessageHandler) OnSendDocument(msg *message.Message, pType string, req *dto.MessageCreatedV1) error {
	p, ok := h.providers[pType]
	if !ok {
		h.logger.Warn("SKIPPED: provider_not_found", "type", pType)
		return nil
	}
	_, err := p.SendDocument(msg.Context(), req.ToDomain())
	return err
}
