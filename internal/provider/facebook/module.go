package facebook

import (
	gatewayv1 "github.com/webitel/im-providers-service/gen/go/gateway/v1"
	imgateway "github.com/webitel/im-providers-service/infra/client/grpc/im-gateway"
	"github.com/webitel/im-providers-service/internal/provider"
	"go.uber.org/fx"
)

var Module = fx.Module("facebook",
	fx.Provide(
		fx.Annotate(
			New,
			fx.As(new(provider.Provider)),
			fx.ResultTags(`group:"providers"`),
		),
	),
	fx.Provide(
		func(p provider.Provider) provider.Sender { return p },
		func(p provider.Provider) provider.Receiver { return p },
	),
	fx.Provide(
		func(client *imgateway.Client) gatewayv1.ContactsClient {
			return client
		},
	),
)
