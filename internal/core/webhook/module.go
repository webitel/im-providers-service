package webhook

import (
	"go.uber.org/fx"
)

// Module exports the webhook handler to the FX graph.
// [FX_MODULE] Self-contained registration for the webhook layer.
var Module = fx.Options(
	fx.Provide(
		fx.Annotate(
			NewHandler,
			// [FX_INJECTION] Map dependencies:
			// 1. slog.Logger (unnamed)
			// 2. Slice of providers from the "providers" value group
			fx.ParamTags(``, `group:"providers"`),
		),
	),
)
