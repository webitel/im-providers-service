package imgateway

import (
	"context"

	gatewayv1 "github.com/webitel/im-providers-service/gen/go/gateway/v1"
	"google.golang.org/grpc"
)

// SendText delivers plain text messages to the core gateway.
func (c *Client) SendText(
	ctx context.Context,
	in *gatewayv1.SendTextRequest,
	opts ...grpc.CallOption,
) (*gatewayv1.SendTextResponse, error) {
	var resp *gatewayv1.SendTextResponse
	err := c.msgRPC.Execute(ctx, func(api gatewayv1.MessageClient) error {
		var err error
		resp, err = api.SendText(ctx, in, opts...)
		return err
	})
	return resp, err
}

// SendFile delivers documents or files to the core gateway.
func (c *Client) SendFile(
	ctx context.Context,
	in *gatewayv1.SendDocumentRequest,
	opts ...grpc.CallOption,
) (*gatewayv1.SendDocumentResponse, error) {
	var resp *gatewayv1.SendDocumentResponse
	err := c.msgRPC.Execute(ctx, func(api gatewayv1.MessageClient) error {
		var err error
		resp, err = api.SendFile(ctx, in, opts...)
		return err
	})
	return resp, err
}

// SendImage delivers image-specific messages to the core gateway.
func (c *Client) SendImage(
	ctx context.Context,
	in *gatewayv1.SendImageRequest,
	opts ...grpc.CallOption,
) (*gatewayv1.SendImageResponse, error) {
	var resp *gatewayv1.SendImageResponse
	err := c.msgRPC.Execute(ctx, func(api gatewayv1.MessageClient) error {
		var err error
		resp, err = api.SendImage(ctx, in, opts...)
		return err
	})
	return resp, err
}
