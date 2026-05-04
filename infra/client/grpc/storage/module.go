package storage

import (
	"context"

	"go.uber.org/fx"
)

// Module manages the lifecycle of the IM Contact gRPC client.
var Module = fx.Module(
	"storage_client",

	fx.Provide(New),

	fx.Invoke(func(lc fx.Lifecycle, client *Client) {
		lc.Append(fx.Hook{
			OnStop: func(ctx context.Context) error {
				return client.Close()
			},
		})
	}),
)
