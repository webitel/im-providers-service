package handler

import (
	"go.uber.org/fx"

	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	grpcsrv "github.com/webitel/im-providers-service/infra/srv/grpc"
)

var Module = fx.Module("instagram/handler",
	fx.Provide(
		NewInstagramHandler,
	),
	fx.Invoke(RegisterInstagramServices),
)

// RegisterInstagramServices connects the Instagram gRPC handlers to the gRPC server.
//
// NOTE: MetaOAuthService is intentionally NOT registered here — it is a single
// gRPC service already owned by facebook.Module, and gRPC panics on duplicate
// service registration. The StartMetaOAuth RPC is keyed by meta_app_id, so a
// single dispatcher that routes by the MetaApp's provider type is the correct
// home for Instagram Business Login OAuth (follow-up). Until then, an Instagram
// gate is created directly via CreateInstagramGate with a pre-obtained token.
func RegisterInstagramServices(
	server *grpcsrv.Server,
	instagram *InstagramHandler,
) {
	impb.RegisterInstagramServiceServer(server.Server, instagram)
}
