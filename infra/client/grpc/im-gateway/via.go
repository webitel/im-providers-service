package imgateway

import (
	"context"

	gatewayv1 "github.com/webitel/im-providers-service/gen/go/gateway/v1"
	"google.golang.org/grpc"
)

func (c *Client) CreateVia(ctx context.Context, in *gatewayv1.ViasServiceCreateRequest, opts ...grpc.CallOption) (*gatewayv1.ViasServiceCreateResponse, error) {
	var resp *gatewayv1.ViasServiceCreateResponse
	err := c.viasRPC.Execute(ctx, func(api gatewayv1.ViasServiceClient) error {
		var err error
		resp, err = api.Create(ctx, in, opts...)
		return err
	})
	return resp, err
}
