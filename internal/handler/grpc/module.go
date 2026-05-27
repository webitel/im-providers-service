package grpc

// Module registers all gRPC handlers and their registration logic.
// var Module = fx.Module("provider-grpc",
// 	fx.Provide(
// 		NewGateHandler,
// 		NewOutboundMessageHandler,
// 		NewFacebookHandler,
// 		NewMetaAppHandler,
// 		NewMetaOauthHandler,
// 	),
// 	fx.Invoke(RegisterProviderServices),
// )

// RegisterProviderServices connects our internal handlers to the actual gRPC server.
// func RegisterProviderServices(
// 	server *grpcsrv.Server,
// 	gate *GateHandler,
// 	outboundMessage *OutboundMessageHandler,
// 	facebook *FacebookHandler,
// 	metaApp *MetaAppHandler,
// 	metaOAuth *MetaOauthHandler,
// 	whatsapp whatsapp.WhatsAppGateServer,
// ) {
// 	// Register each service defined in your proto files
// 	impb.RegisterGateServiceServer(server.Server, gate)
// 	impb.RegisterFacebookServiceServer(server.Server, facebook)
// 	impb.RegisterMetaAppServiceServer(server.Server, metaApp)
// 	impb.RegisterMetaOAuthServiceServer(server.Server, metaOAuth)
// 	impb.RegisterWhatsAppServiceServer(server.Server, whatsapp)
// 	impb.RegisterProviderMessageServiceServer(server.Server, outboundMessage)
// }
