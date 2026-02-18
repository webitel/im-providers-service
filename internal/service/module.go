package service

import "go.uber.org/fx"

// Module defines the messaging and authorization service logic.
var Module = fx.Module("service",
	fx.Provide(
		// [MESSENGER]
		fx.Annotate(
			NewMessageService,
			fx.As(new(Messenger)),
		),
		// [AUTHER]
		fx.Annotate(
			NewAuthService,
			fx.As(new(Auther)),
		),
	),
)
