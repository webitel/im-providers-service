package facebook

import (
	"github.com/webitel/im-providers-service/internal/provider"
	"go.uber.org/fx"
)

// Module provides the Facebook adapter to the application.
// [FX_ADAPTER_REGISTRATION]
var Module = fx.Provide(
	fx.Annotate(
		New,
		// [STRATEGY] Provide as ALL relevant interfaces
		fx.As(new(provider.Sender)),   // For AMQP outbound
		fx.As(new(provider.Receiver)), // For HTTP webhooks
		fx.As(new(provider.Provider)), // For general use

		// Add to the common group for automatic discovery
		fx.ResultTags(`group:"providers"`),
	),
)
