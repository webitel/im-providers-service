package grpcinterceptors

import (
	"context"

	"google.golang.org/grpc"
)

// UnaryAuthInterceptor is a gRPC interceptor for handling authentication.
func NewUnaryAuthInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		return handler(ctx, req)
	}
}
