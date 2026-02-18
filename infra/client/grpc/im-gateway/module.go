package imgateway

import (
	"context"

	"go.uber.org/fx"
)

// Module manages the lifecycle of all IM Gateway gRPC clients.
var Module = fx.Module(
	"imgateway_client",

	// [CONSTRUCTOR] Provides the unified gateway client
	fx.Provide(New),

	// [LIFECYCLE] Ensures all gRPC connection pools are closed gracefully
	fx.Invoke(func(lc fx.Lifecycle, client *Client) {
		lc.Append(fx.Hook{
			OnStop: func(ctx context.Context) error {
				// [CLEANUP] Close all internal connection pools
				// Assuming your Client struct manages these 3 connections

				if err := client.msgRPC.Close(); err != nil {
					return err
				}

				if err := client.accountRPC.Close(); err != nil {
					return err
				}

				if err := client.contactRPC.Close(); err != nil {
					return err
				}

				return nil
			},
		})
	}),
)
