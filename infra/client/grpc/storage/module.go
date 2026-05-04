package storage

import (
	"context"

	"go.uber.org/fx"
)

var Module = fx.Module(
	"storage_client",
	fx.Provide(New),
	fx.Invoke(func(lc fx.Lifecycle, client *Client) {
		lc.Append(fx.Hook{
			OnStop: func(ctx context.Context) error {
				return client.rpc.Close()
			},
		})
	}),
)
