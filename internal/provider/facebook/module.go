package facebook

import (
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
)
