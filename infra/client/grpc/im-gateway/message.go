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
func (c *Client) SendDocument(
	ctx context.Context,
	in *gatewayv1.SendDocumentRequest,
	opts ...grpc.CallOption,
) (*gatewayv1.SendDocumentResponse, error) {
	var resp *gatewayv1.SendDocumentResponse
	err := c.msgRPC.Execute(ctx, func(api gatewayv1.MessageClient) error {
		var err error
		resp, err = api.SendDocument(ctx, in, opts...)
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

// Read implements [gateway.MessageClient].
func (c *Client) Read(ctx context.Context, in *gatewayv1.ReadMessageRequest, opts ...grpc.CallOption) (*gatewayv1.ReadMessageResponse, error) {
	panic("unimplemented")
}

// SendContact implements [gateway.MessageClient].
func (c *Client) SendContact(ctx context.Context, in *gatewayv1.SendContactRequest, opts ...grpc.CallOption) (*gatewayv1.SendMessageResponse, error) {
	var resp *gatewayv1.SendMessageResponse
	err := c.msgRPC.Execute(ctx, func(mc gatewayv1.MessageClient) error {
		var err error
		resp, err = mc.SendContact(ctx, in)
		return err
	})

	return resp, err
}

// SendInteractive implements [gateway.MessageClient].
func (c *Client) SendInteractive(ctx context.Context, in *gatewayv1.SendInteractiveMessageRequest, opts ...grpc.CallOption) (*gatewayv1.SendMessageResponse, error) {
	var resp *gatewayv1.SendMessageResponse
	err := c.msgRPC.Execute(ctx, func(api gatewayv1.MessageClient) error {
		var err error
		resp, err = api.SendInteractive(ctx, in, opts...)
		return err
	})
	return resp, err
}

// SendInteractiveCallback implements [gateway.MessageClient].
func (c *Client) SendInteractiveCallback(ctx context.Context, in *gatewayv1.InteractiveCallbackRequest, opts ...grpc.CallOption) (*gatewayv1.InteractiveCallbackResponse, error) {
	var resp *gatewayv1.InteractiveCallbackResponse
	err := c.msgRPC.Execute(ctx, func(api gatewayv1.MessageClient) error {
		var err error
		resp, err = api.SendInteractiveCallback(ctx, in, opts...)
		return err
	})
	return resp, err
}

// SendLocation implements [gateway.MessageClient].
func (c *Client) SendLocation(ctx context.Context, in *gatewayv1.SendLocationRequest, opts ...grpc.CallOption) (*gatewayv1.SendMessageResponse, error) {
	var resp *gatewayv1.SendMessageResponse
	err := c.msgRPC.Execute(ctx, func(mc gatewayv1.MessageClient) error {
		var err error
		resp, err = mc.SendLocation(ctx, in)
		return err
	})

	return resp, err
}

// SendSystemMessage implements [gateway.MessageClient].
func (c *Client) SendSystemMessage(ctx context.Context, in *gatewayv1.SendSystemMessageRequest, opts ...grpc.CallOption) (*gatewayv1.SendMessageResponse, error) {
	panic("unimplemented")
}
