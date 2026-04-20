package imauth

import (
	"context"
	"fmt"
	"log/slog"

	authv1 "github.com/webitel/im-providers-service/gen/go/auth/v1"
	webitel "github.com/webitel/im-providers-service/infra/client/grpc"
	infratls "github.com/webitel/im-providers-service/infra/tls"
	"github.com/webitel/webitel-go-kit/infra/discovery"
	rpc "github.com/webitel/webitel-go-kit/infra/transport/gRPC"
	"google.golang.org/grpc"
)

const ServiceName string = "im-account-service"

// [INTERFACE_GUARD] Now correctly matches the CLIENT interface.
var _ authv1.AccountClient = (*Client)(nil)

type Client struct {
	logger *slog.Logger
	rpc    *rpc.Client[authv1.AccountClient]
	tls    *infratls.Config
}

// New initializes a resilient gRPC client for the Auth service.
func New(logger *slog.Logger, discovery discovery.DiscoveryProvider, tls *infratls.Config) (*Client, error) {
	factory := func(conn *grpc.ClientConn) authv1.AccountClient {
		return authv1.NewAccountClient(conn)
	}

	c, err := webitel.New(logger, discovery, ServiceName, tls, factory)
	if err != nil {
		return nil, fmt.Errorf("[im-auth-client] initialization failed: %w", err)
	}

	return &Client{
		logger: logger,
		rpc:    c,
	}, nil
}

// Inspect validates the access token.
func (c *Client) Inspect(ctx context.Context, in *authv1.InspectRequest, opts ...grpc.CallOption) (*authv1.Authorization, error) {
	var resp *authv1.Authorization
	err := c.rpc.Execute(ctx, func(api authv1.AccountClient) error {
		var err error
		resp, err = api.Inspect(ctx, in, opts...)
		return err
	})
	return resp, err
}

func (c *Client) Token(ctx context.Context, in *authv1.TokenRequest, opts ...grpc.CallOption) (*authv1.Authorization, error) {
	var resp *authv1.Authorization
	err := c.rpc.Execute(ctx, func(api authv1.AccountClient) error {
		var err error
		resp, err = api.Token(ctx, in, opts...)
		return err
	})
	return resp, err
}

func (c *Client) Logout(ctx context.Context, in *authv1.LogoutRequest, opts ...grpc.CallOption) (*authv1.LogoutResponse, error) {
	var resp *authv1.LogoutResponse
	err := c.rpc.Execute(ctx, func(api authv1.AccountClient) error {
		var err error
		resp, err = api.Logout(ctx, in, opts...)
		return err
	})
	return resp, err
}

func (c *Client) RegisterDevice(ctx context.Context, in *authv1.RegisterDeviceRequest, opts ...grpc.CallOption) (*authv1.RegisterDeviceResponse, error) {
	var resp *authv1.RegisterDeviceResponse
	err := c.rpc.Execute(ctx, func(api authv1.AccountClient) error {
		var err error
		resp, err = api.RegisterDevice(ctx, in, opts...)
		return err
	})
	return resp, err
}

func (c *Client) UnregisterDevice(ctx context.Context, in *authv1.UnregisterDeviceRequest, opts ...grpc.CallOption) (*authv1.UnregisterDeviceResponse, error) {
	var resp *authv1.UnregisterDeviceResponse
	err := c.rpc.Execute(ctx, func(api authv1.AccountClient) error {
		var err error
		resp, err = api.UnregisterDevice(ctx, in, opts...)
		return err
	})
	return resp, err
}

// GetAuthorizations implements [auth.AccountClient].
func (c *Client) GetAuthorizations(ctx context.Context, in *authv1.GetAuthorizationRequest, opts ...grpc.CallOption) (*authv1.AuthorizationList, error) {
	var resp *authv1.AuthorizationList
	err := c.rpc.Execute(ctx, func(api authv1.AccountClient) error {
		var err error
		resp, err = api.GetAuthorizations(ctx, in, opts...)
		return err
	})
	return resp, err
}

func (c *Client) Close() error {
	if c.rpc != nil {
		return c.rpc.Close()
	}
	return nil
}
