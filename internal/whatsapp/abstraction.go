package whatsapp

import (
	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/im-providers-service/internal/provider"
	"github.com/webitel/im-providers-service/internal/whatsapp/messaging"
	"github.com/webitel/im-providers-service/internal/whatsapp/webhook"
)

type WhatsAppGateServer interface {
	impb.WhatsAppServiceServer
}

type WhatsApp struct {
	*webhook.WebhookManager
	*messaging.Messaging
}

var _ provider.CapabilityReporter = (*WhatsApp)(nil)

// Capabilities: WhatsApp Cloud API emits statuses webhooks with
// sent/delivered/read/failed states per message.
func (w *WhatsApp) Capabilities() sharedmodel.ProviderCapabilities {
	return sharedmodel.ProviderCapabilities{
		SupportsDelivered: true,
		SupportsRead:      true,
		SupportsFailed:    true,
	}
}
