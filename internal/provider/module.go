package provider

import "go.uber.org/fx"

// Module ensures the Registry is provided with all adapters from the "providers" group.
var Module = fx.Module("provider",
	fx.Provide(
		fx.Annotate(
			NewRegistry,
			fx.ParamTags(`group:"providers"`),
		),
	),
)
