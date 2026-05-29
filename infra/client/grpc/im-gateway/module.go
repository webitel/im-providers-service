package imgateway

import (
	"context"

	"go.uber.org/fx"
)

var Module = fx.Module(
	"imgateway_client",
	fx.Provide(New),
	fx.Invoke(func(lc fx.Lifecycle, client *Client) {
		lc.Append(fx.Hook{
			OnStop: func(ctx context.Context) error {
				if err := client.msgRPC.Close(); err != nil {
					return err
				}

				if err := client.accountRPC.Close(); err != nil {
					return err
				}

				if err := client.contactRPC.Close(); err != nil {
					return err
				}

				if err := client.viasRPC.Close(); err != nil {
					return err
				}

				return nil
			},
		})
	}),
)
