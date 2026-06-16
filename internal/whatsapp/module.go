package whatsapp

import (
	"context"
	"log/slog"

	"github.com/webitel/im-providers-service/config"
	imgateway "github.com/webitel/im-providers-service/infra/client/grpc/im-gateway"
	"github.com/webitel/im-providers-service/infra/db/postgresx"
	"github.com/webitel/im-providers-service/internal/core/service"
	"github.com/webitel/im-providers-service/internal/provider"
	"github.com/webitel/im-providers-service/internal/whatsapp/gate"
	"github.com/webitel/im-providers-service/internal/whatsapp/messaging"
	"github.com/webitel/im-providers-service/internal/whatsapp/resolver"
	"github.com/webitel/im-providers-service/internal/whatsapp/webhook"
	"github.com/webitel/im-providers-service/pkg/crypto"
	"github.com/webitel/webitel-go-kit/pkg/semconv"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"whatsapp",
	fx.Provide(ProvideNewPostgresxConnection),
	fx.Provide(
		func(logger *slog.Logger, db postgresx.DB, internalContactResolver *imgateway.Client, encryptor crypto.Encryptor) WhatsAppGateServer {
			gateWire := gate.NewGateModule(logger, db, internalContactResolver, encryptor)
			return gateWire.GateServer
		},
	),

	fx.Provide(
		fx.Annotate(
			func(
				logger *slog.Logger,
				db postgresx.DB,
				encryptor crypto.Encryptor,
				coreMessanger service.Messenger,
				client *imgateway.Client,
				media *service.MediaService,
			) *WhatsApp {
				webhookResolver := resolver.NewResolverModule[*webhook.WhatsAppBusinessAccountResolveQuery](logger, db)

				webhookConfig := webhook.WebhookManagerConfig{
					Logger: logger,
				}

				webhhokModule, err := webhook.NewWebhookModule(webhookConfig, encryptor, coreMessanger, webhookResolver.Resolver, client, media)
				if err != nil {
					logger.Error("whatsapp:wire:constructing new webhook module", semconv.ErrorKey, err)
					return nil
				}

				whatsAppMessagingClient := messaging.NewMessagingWire(
					logger,
					encryptor,
					client,
					db,
				)

				return &WhatsApp{
					WebhookManager: webhhokModule.WebhookManager,
					Messaging:      whatsAppMessagingClient.Messaging,
				}
			},
			fx.As(new(provider.Provider)),
			fx.ResultTags(`group:"providers"`),
		),
	),
)

func ProvideNewPostgresxConnection(cfg *config.Config, l *slog.Logger, lc fx.Lifecycle) (postgresx.DB, error) {
	ctx := context.Background()
	db, err := postgresx.New(ctx, cfg.Postgres.DSN, cfg.Postgres.ToOpenOptions()...)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			db.Close()
			return nil
		},
	})

	return db, err
}
