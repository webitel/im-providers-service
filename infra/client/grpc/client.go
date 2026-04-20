package client

import (
	"context"
	"fmt"
	"log/slog"

	infratls "github.com/webitel/im-providers-service/infra/tls"
	ds "github.com/webitel/webitel-go-kit/infra/discovery"
	rpc "github.com/webitel/webitel-go-kit/infra/transport/gRPC"
	"github.com/webitel/webitel-go-kit/infra/transport/gRPC/resolver/discovery"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// New initializes a go-kit RPC client with Discovery and OPTIONAL Circuit Breaker.
// [CHANGE] Added 'withBreaker' boolean parameter.
func New[T any](
	log *slog.Logger,
	dp ds.DiscoveryProvider,
	target string,
	tlsCong *infratls.Config,
	factory rpc.ClientFactory[T],
) (*rpc.Client[T], error) {
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(tlsCong.Client)),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithResolvers(discovery.NewBuilder(dp, discovery.WithInsecure(true))),
	}

	client, err := rpc.NewClient(
		context.Background(),
		factory,
		rpc.WithTarget(fmt.Sprintf("discovery:///%s", target)),
		rpc.WithDialOptions(options...),
		// [RETRY] Built-in transport-level retries (works regardless of breaker)
		rpc.WithRetry(rpc.DefaultRetryConfig()),
	)
	if err != nil {
		return nil, err
	}

	return client, nil
}
