package provider

import "go.uber.org/fx"

// Module wires the Registry, collecting all adapters from the "providers" fx group.
var Module = fx.Module("provider",
	fx.Provide(
		fx.Annotate(
			NewRegistry,
			fx.ParamTags(`group:"providers"`),
		),
	),
)
