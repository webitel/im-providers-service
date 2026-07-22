// Package imthread wraps the im-thread-service MessageStatus gRPC API used
// to report per-recipient delivery statuses (delivered/read/failed) resolved
// from provider webhook receipts and synchronous send failures.
package imthread

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/webitel/webitel-go-kit/infra/discovery"
	rpc "github.com/webitel/webitel-go-kit/infra/transport/gRPC"
	"google.golang.org/grpc"

	threadv1 "github.com/webitel/im-providers-service/gen/go/thread/v1"
	webitel "github.com/webitel/im-providers-service/infra/client/grpc"
	infratls "github.com/webitel/im-providers-service/infra/tls"
)

const ServiceName string = "im-thread-service"

type Client struct {
	logger    *slog.Logger
	statusRPC *rpc.Client[threadv1.MessageStatusClient]
}

// New initializes a resilient gRPC client for the im-thread MessageStatus service.
func New(
	logger *slog.Logger,
	discovery discovery.DiscoveryProvider,
	tls *infratls.Config,
) (*Client, error) {
	statusClient, err := webitel.New(
		logger,
		discovery,
		ServiceName,
		tls,
		func(conn *grpc.ClientConn) threadv1.MessageStatusClient {
			return threadv1.NewMessageStatusClient(conn)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("[im-thread-client] message status init failed: %w", err)
	}

	return &Client{
		logger:    logger.With(slog.String("component", "im-thread-client")),
		statusRPC: statusClient,
	}, nil
}

func (c *Client) MarkDelivered(ctx context.Context, in *threadv1.MarkDeliveredRequest, opts ...grpc.CallOption) (*threadv1.MarkStatusResponse, error) {
	var resp *threadv1.MarkStatusResponse

	err := c.statusRPC.Execute(ctx, func(api threadv1.MessageStatusClient) error {
		var err error
		resp, err = api.MarkDelivered(ctx, in, opts...)

		return err
	})

	return resp, err
}

func (c *Client) MarkRead(ctx context.Context, in *threadv1.MarkReadRequest, opts ...grpc.CallOption) (*threadv1.MarkStatusResponse, error) {
	var resp *threadv1.MarkStatusResponse

	err := c.statusRPC.Execute(ctx, func(api threadv1.MessageStatusClient) error {
		var err error
		resp, err = api.MarkRead(ctx, in, opts...)

		return err
	})

	return resp, err
}

func (c *Client) MarkFailed(ctx context.Context, in *threadv1.MarkFailedRequest, opts ...grpc.CallOption) (*threadv1.MarkStatusResponse, error) {
	var resp *threadv1.MarkStatusResponse

	err := c.statusRPC.Execute(ctx, func(api threadv1.MessageStatusClient) error {
		var err error
		resp, err = api.MarkFailed(ctx, in, opts...)

		return err
	})

	return resp, err
}

// Close gracefully shuts down the underlying gRPC connection pool.
func (c *Client) Close() error {
	if c.statusRPC != nil {
		return c.statusRPC.Close()
	}

	return nil
}
