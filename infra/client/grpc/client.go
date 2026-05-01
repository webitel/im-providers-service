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
	"google.golang.org/grpc/metadata"
)

func New[T any](
	log *slog.Logger,
	dp ds.DiscoveryProvider,
	target string,
	tlsConfig *infratls.Config,
	factory rpc.ClientFactory[T],
) (*rpc.Client[T], error) {
	authInterceptor := func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.MD{}
		}

		// Fixed type for provider-to-gateway communication
		md.Set("x-webitel-type", "provider")

		// If an IdentityProvider is found in context, set the provider header
		if p, ok := GetIdentity(ctx); ok {
			md.Set("x-webitel-provider", p.Identity())
		}

		return invoker(metadata.NewOutgoingContext(ctx, md), method, req, reply, cc, opts...)
	}

	options := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig.Client)),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithResolvers(discovery.NewBuilder(dp, discovery.WithInsecure(true))),
		grpc.WithChainUnaryInterceptor(authInterceptor),
	}

	client, err := rpc.NewClient(
		context.Background(),
		factory,
		rpc.WithTarget(fmt.Sprintf("discovery:///%s", target)),
		rpc.WithDialOptions(options...),
		rpc.WithRetry(rpc.DefaultRetryConfig()),
	)
	if err != nil {
		return nil, err
	}

	return client, nil
}
