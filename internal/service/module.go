package service

import "go.uber.org/fx"

// Module defines the messaging, authorization, and gate management service logic.
var Module = fx.Module("service",
	fx.Provide(
		// [GATES] Unified gateway management
		fx.Annotate(
			NewGateService,
			fx.As(new(GateManager)),
		),

		// [FACEBOOK] Facebook-specific logic
		fx.Annotate(
			NewFacebookService,
			fx.As(new(FacebookManager)),
		),

		// [META APP] Meta application configuration management
		fx.Annotate(
			NewMetaAppService,
			fx.As(new(MetaAppManager)),
		),

		// [META OAUTH] Handling Meta OAuth2 flows
		fx.Annotate(
			NewMetaOAuthService,
			fx.As(new(MetaOAuthManager)),
		),

		// [WHATSAPP] WhatsApp-specific logic (placeholder for your whatsapp.go)
		fx.Annotate(
			NewWhatsAppService,
			fx.As(new(WhatsAppManager)),
		),

		// [MESSENGER] General message processing
		fx.Annotate(
			NewMessageService,
			fx.As(new(Messenger)),
		),

		// [AUTHER] Authentication and authorization
		fx.Annotate(
			NewAuthService,
			fx.As(new(Auther)),
		),
	),
)
