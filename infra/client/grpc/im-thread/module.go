package imthread

import (
	"context"

	"go.uber.org/fx"
)

var Module = fx.Module(
	"imthread_client",
	fx.Provide(New),
	fx.Invoke(func(lc fx.Lifecycle, client *Client) {
		lc.Append(fx.Hook{
			OnStop: func(_ context.Context) error {
				return client.Close()
			},
		})
	}),
)
