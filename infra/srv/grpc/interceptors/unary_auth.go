package interceptors

import (
	"context"

	"github.com/webitel/im-providers-service/infra/auth"
	"google.golang.org/grpc"
)

// NewUnaryAuthInterceptor provides identification for standard RPC calls
func NewUnaryAuthInterceptor(authorizer auth.Authorizer) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		newCtx, err := authorizer.SetIdentity(ctx)
		if err != nil {
			return nil, err
		}

		return handler(newCtx, req)
	}
}
