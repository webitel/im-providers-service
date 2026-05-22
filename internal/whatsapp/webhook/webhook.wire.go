package webhook

import (
	imgateway "github.com/webitel/im-providers-service/infra/client/grpc/im-gateway"
	"github.com/webitel/im-providers-service/internal/service"
	"github.com/webitel/im-providers-service/internal/whatsapp/common"
)

type webhookModule struct {
	WebhookManager *WebhookManager
}

func NewWebhookModule(
	config WebhookManagerConfig,
	encryptor common.Encryptor,
	coreMessanger CoreMessanger,
	whatsAppBusinessAccountResolver WhatsAppBusinessAccountResolver,
	client *imgateway.Client,
	media service.MediaManager,
) (*webhookModule, error) {
	var (
		coreMessangerDecorated = newDecoratedCoreMessanger(coreMessanger, client)
		webhookUsecase         = newWebhook(config.Logger, coreMessangerDecorated, whatsAppBusinessAccountResolver, encryptor, media)
	)

	webhookMaanager, err := newWebhookManager(config, webhookUsecase)
	if err != nil {
		return nil, err
	}

	return &webhookModule{
		WebhookManager: webhookMaanager,
	}, nil
}
