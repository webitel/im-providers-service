// package service

// import "go.uber.org/fx"

// var Module = fx.Module("service",
// 	fx.Provide(
// 		NewMediaService,
// 		fx.Annotate(NewGateService, fx.As(new(GateManager))),
// 		fx.Annotate(NewFacebookService, fx.As(new(FacebookManager))),
// 		fx.Annotate(NewMetaAppService, fx.As(new(MetaAppManager))),
// 		fx.Annotate(NewMetaOAuthService, fx.As(new(MetaOAuthManager))),
// 		fx.Annotate(NewWhatsAppService, fx.As(new(WhatsAppManager))),
// 		fx.Annotate(NewAuthService, fx.As(new(Auther))),
// 		fx.Annotate(NewMediaService, fx.As(new(MediaManager))),

// 		fx.Annotate(
// 			NewMessageService,
// 			fx.ResultTags(`name:"baseMessenger"`),
// 		),
// 		fx.Annotate(
// 			ProvideDecoratedMessenger,
// 			fx.ParamTags(`name:"baseMessenger"`),
// 			fx.As(new(Messenger)),
// 		),
// 	),
// )

// func ProvideDecoratedMessenger(baseMessenger Messenger) Messenger {
// 	return NewMessengerAuthMiddleware(baseMessenger)
// }
