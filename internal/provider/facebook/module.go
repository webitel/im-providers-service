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
		fx.As(new(provider.Provider)),
		fx.ResultTags(`group:"providers"`),
	),
)
