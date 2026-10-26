package interceptors

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	werrors "github.com/webitel/webitel-go-kit/pkg/errors"
	intrcp "github.com/webitel/webitel-go-kit/pkg/interceptors"
)

// TestStatusCodeSurvivesErrorInterceptor runs the real go-kit error interceptor
// around ours, the same order as the server chain.
func TestStatusCodeSurvivesErrorInterceptor(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		{"status not found", status.Error(codes.NotFound, "gate not found"), codes.NotFound},
		{"status invalid argument", status.Error(codes.InvalidArgument, "id must be a UUID"), codes.InvalidArgument},
		{"status unauthenticated", status.Error(codes.Unauthenticated, "no token"), codes.Unauthenticated},
		{"go-kit error keeps its code", werrors.NotFound("gate not found"), codes.NotFound},
		{"plain error stays internal", errors.New("db down"), codes.Internal},
		{"status unknown stays internal", status.Error(codes.Unknown, "boom"), codes.Internal},
	}

	errIntrcp := intrcp.UnaryServerErrorInterceptor()
	ours := NewUnaryStatusCodeInterceptor()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := func(context.Context, any) (any, error) { return nil, tt.err }
			chained := func(ctx context.Context, req any) (any, error) {
				return ours(ctx, req, info, handler)
			}

			_, err := errIntrcp(context.Background(), nil, info, chained)
			if got := status.Code(err); got != tt.wantCode {
				t.Fatalf("code = %v, want %v (err: %v)", got, tt.wantCode, err)
			}
		})
	}
}
