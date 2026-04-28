package imgateway

import (
	"context"

	gatewayv1 "github.com/webitel/im-providers-service/gen/go/gateway/v1"
	"google.golang.org/grpc"
)

func (c *Client) Search(
	ctx context.Context,
	in *gatewayv1.SearchContactRequest,
	opts ...grpc.CallOption,
) (*gatewayv1.ContactList, error) {
	var resp *gatewayv1.ContactList
	err := c.contactRPC.Execute(ctx, func(api gatewayv1.ContactsClient) error {
		var err error
		resp, err = api.Search(ctx, in, opts...)
		return err
	})
	return resp, err
}

func (c *Client) Create(ctx context.Context, in *gatewayv1.CreateContactRequest, opts ...grpc.CallOption) (*gatewayv1.Contact, error) {
	var resp *gatewayv1.Contact
	err := c.contactRPC.Execute(ctx, func(api gatewayv1.ContactsClient) error {
		var err error
		resp, err = api.Create(ctx, in, opts...)
		return err
	})
	return resp, err
}

// Locate implements [gateway.ContactsClient].
func (c *Client) Locate(ctx context.Context, in *gatewayv1.LocateConatctRequest, opts ...grpc.CallOption) (*gatewayv1.LocateContactResponse, error) {
	var resp *gatewayv1.LocateContactResponse
	err := c.contactRPC.Execute(ctx, func(api gatewayv1.ContactsClient) error {
		var err error
		resp, err = api.Locate(ctx, in, opts...)
		return err
	})
	return resp, err
}
