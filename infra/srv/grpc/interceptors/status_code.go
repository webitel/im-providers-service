package interceptors

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	werrors "github.com/webitel/webitel-go-kit/pkg/errors"
)

// NewUnaryStatusCodeInterceptor carries plain gRPC status codes over to the
// go-kit error interceptor. That interceptor reads the code only from go-kit
// errors and reports every status.Error (handlers, auth, protovalidate) as
// Internal, so NotFound / InvalidArgument would surface as HTTP 500.
// Must sit directly inside intrcp.UnaryServerErrorInterceptor in the chain.
func NewUnaryStatusCodeInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		if err == nil {
			return resp, nil
		}

		return resp, withStatusCode(err)
	}
}

func withStatusCode(err error) error {
	if _, ok := werrors.Value(err, werrors.ErrKeyCode).(codes.Code); ok {
		return err
	}

	s, ok := status.FromError(err)
	if !ok || s.Code() == codes.Unknown {
		return err
	}

	return werrors.New(s.Message(), werrors.WithCode(s.Code()), werrors.WithCause(err))
}
