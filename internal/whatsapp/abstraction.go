package whatsapp

import (
	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
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
