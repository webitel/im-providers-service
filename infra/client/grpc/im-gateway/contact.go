package imgateway

import (
	"context"

	gatewayv1 "github.com/webitel/im-providers-service/gen/go/gateway/v1"
	"google.golang.org/grpc"
)

// Search finds contacts within the gateway registry.
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
