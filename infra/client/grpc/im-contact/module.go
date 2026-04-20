package imcontact

import (
	"context"

	"go.uber.org/fx"
)

// Module manages the lifecycle of the IM Contact gRPC client.
var Module = fx.Module(
	"imcontact_client",

	// [CONSTRUCTOR] Provides the contact service client
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
