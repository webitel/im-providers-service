package imgateway

import (
	"fmt"
	"log/slog"

	gatewayv1 "github.com/webitel/im-providers-service/gen/go/gateway/v1"
	webitel "github.com/webitel/im-providers-service/infra/client/grpc"
	"github.com/webitel/webitel-go-kit/infra/discovery"
	rpc "github.com/webitel/webitel-go-kit/infra/transport/gRPC"
	"google.golang.org/grpc"
)

const ServiceName string = "im-gateway-service"

// [INTERFACE_GUARD] Ensure Client implements all necessary proto interfaces.
var (
	_ gatewayv1.MessageClient  = (*Client)(nil)
	_ gatewayv1.AccountClient  = (*Client)(nil)
	_ gatewayv1.ContactsClient = (*Client)(nil)
)

type Client struct {
	logger     *slog.Logger
	msgRPC     *rpc.Client[gatewayv1.MessageClient]
	accountRPC *rpc.Client[gatewayv1.AccountClient]
	contactRPC *rpc.Client[gatewayv1.ContactsClient]
}

// New initializes a resilient gRPC client for the IM Gateway service.
func New(
	logger *slog.Logger,
	discovery discovery.DiscoveryProvider,
) (*Client, error) {
	// Initialize Message Client
	msg, err := webitel.New(
		logger,
		discovery,
		ServiceName,
		func(conn *grpc.ClientConn) gatewayv1.MessageClient {
			return gatewayv1.NewMessageClient(conn)
		})
	if err != nil {
		return nil, fmt.Errorf("[im-gateway-client] messages init failed: %w", err)
	}

	// Initialize Account Client
	acc, err := webitel.New(
		logger,
		discovery,
		ServiceName,
		func(conn *grpc.ClientConn) gatewayv1.AccountClient {
			return gatewayv1.NewAccountClient(conn)
		})
	if err != nil {
		return nil, fmt.Errorf("[im-gateway-client] account init failed: %w", err)
	}

	// Initialize Contacts Client
	cnt, err := webitel.New(
		logger,
		discovery,
		ServiceName,
		func(conn *grpc.ClientConn) gatewayv1.ContactsClient {
			return gatewayv1.NewContactsClient(conn)
		})
	if err != nil {
		return nil, fmt.Errorf("[im-gateway-client] contacts init failed: %w", err)
	}

	return &Client{
		logger:     logger,
		msgRPC:     msg,
		accountRPC: acc,
		contactRPC: cnt,
	}, nil
}
