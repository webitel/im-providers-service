package bot

import (
	"go.uber.org/fx"

	"github.com/webitel/im-providers-service/config"
	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	grpcsrv "github.com/webitel/im-providers-service/infra/srv/grpc"
	"github.com/webitel/im-providers-service/internal/provider"
	tgclient "github.com/webitel/im-providers-service/internal/telegram/bot/client"
	tghandler "github.com/webitel/im-providers-service/internal/telegram/bot/handler"
	tgservice "github.com/webitel/im-providers-service/internal/telegram/bot/service"
	tgstore "github.com/webitel/im-providers-service/internal/telegram/bot/store"
	tgpostgres "github.com/webitel/im-providers-service/internal/telegram/bot/store/postgres"
)

var Module = fx.Module("telegram_bot",
	fx.Provide(
		// Telegram Bot API client, used by the gate-management service to
		// register/unregister webhooks on create/update/delete.
		tgclient.NewClient,

		provideOptions,

		// Provider adapter
		fx.Annotate(
			New,
			fx.As(new(provider.Provider)),
			fx.ResultTags(`group:"providers"`),
		),

		// Store
		fx.Annotate(tgpostgres.NewTelegramBotStore, fx.As(new(tgstore.TelegramBotStore))),

		// Service
		fx.Annotate(tgservice.NewTelegramBotService, fx.As(new(tghandler.TelegramBotService))),

		// gRPC handler
		tghandler.NewTelegramBotHandler,
	),
	fx.Invoke(RegisterTelegramBotServices),
)

// provideOptions carries the config values TelegramBotService needs to build the
// webhook URL registered with Telegram's setWebhook.
func provideOptions(cfg *config.Config) tgservice.Options {
	return tgservice.Options{
		PublicURL:   cfg.Service.PublicURL,
		WebhookPath: cfg.Service.WebhookPath,
	}
}

// RegisterTelegramBotServices connects the Telegram Bot gRPC handler to the gRPC server.
func RegisterTelegramBotServices(server *grpcsrv.Server, telegramBot *tghandler.TelegramBotHandler) {
	impb.RegisterTelegramBotServiceServer(server.Server, telegramBot)
}
