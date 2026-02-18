package imgateway

import (
	"context"

	gatewayv1 "github.com/webitel/im-providers-service/gen/go/gateway/v1"
	"google.golang.org/grpc"
)

// Token generates or refreshes access tokens.
func (c *Client) Token(
	ctx context.Context,
	in *gatewayv1.TokenRequest,
	opts ...grpc.CallOption,
) (*gatewayv1.Authorization, error) {
	var resp *gatewayv1.Authorization
	err := c.accountRPC.Execute(ctx, func(api gatewayv1.AccountClient) error {
		var err error
		resp, err = api.Token(ctx, in, opts...)
		return err
	})
	return resp, err
}

// Inspect current Authorization credentials.
func (c *Client) Inspect(
	ctx context.Context,
	in *gatewayv1.InspectRequest,
	opts ...grpc.CallOption,
) (*gatewayv1.Authorization, error) {
	var resp *gatewayv1.Authorization
	err := c.accountRPC.Execute(ctx, func(api gatewayv1.AccountClient) error {
		var err error
		resp, err = api.Inspect(ctx, in, opts...)
		return err
	})
	return resp, err
}

// Logout terminates the device session.
func (c *Client) Logout(
	ctx context.Context,
	in *gatewayv1.LogoutRequest,
	opts ...grpc.CallOption,
) (*gatewayv1.LogoutResponse, error) {
	var resp *gatewayv1.LogoutResponse
	err := c.accountRPC.Execute(ctx, func(api gatewayv1.AccountClient) error {
		var err error
		resp, err = api.Logout(ctx, in, opts...)
		return err
	})
	return resp, err
}

// RegisterDevice enables PUSH notifications for a specific device.
func (c *Client) RegisterDevice(
	ctx context.Context,
	in *gatewayv1.RegisterDeviceRequest,
	opts ...grpc.CallOption,
) (*gatewayv1.RegisterDeviceResponse, error) {
	var resp *gatewayv1.RegisterDeviceResponse
	err := c.accountRPC.Execute(ctx, func(api gatewayv1.AccountClient) error {
		var err error
		resp, err = api.RegisterDevice(ctx, in, opts...)
		return err
	})
	return resp, err
}

// UnregisterDevice stops PUSH notifications for a device.
func (c *Client) UnregisterDevice(
	ctx context.Context,
	in *gatewayv1.UnregisterDeviceRequest,
	opts ...grpc.CallOption,
) (*gatewayv1.UnregisterDeviceResponse, error) {
	var resp *gatewayv1.UnregisterDeviceResponse
	err := c.accountRPC.Execute(ctx, func(api gatewayv1.AccountClient) error {
		var err error
		resp, err = api.UnregisterDevice(ctx, in, opts...)
		return err
	})
	return resp, err
}
