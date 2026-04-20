package imauth

import (
	"context"

	"go.uber.org/fx"
)

// Module manages the lifecycle of the IM Auth (Account) gRPC client.
var Module = fx.Module(
	"imauth_client",

	// [CONSTRUCTOR] Provides the auth service client
	fx.Provide(New),

	// [LIFECYCLE] Ensures the connection pool is closed gracefully
	fx.Invoke(func(lc fx.Lifecycle, client *Client) {
		lc.Append(fx.Hook{
			OnStop: func(ctx context.Context) error {
				return client.Close()
			},
		})
	}),
)
