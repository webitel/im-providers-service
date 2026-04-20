package imcontact

import (
	"context"
	"fmt"
	"log/slog"

	contactv1 "github.com/webitel/im-providers-service/gen/go/contact/v1"
	webitel "github.com/webitel/im-providers-service/infra/client/grpc"
	infratls "github.com/webitel/im-providers-service/infra/tls"
	"github.com/webitel/webitel-go-kit/infra/discovery"
	rpc "github.com/webitel/webitel-go-kit/infra/transport/gRPC"
	"google.golang.org/grpc"
)

const ServiceName string = "im-contact-service"

type Client struct {
	logger *slog.Logger
	// [GENERIC_RPC] Holds the go-kit RPC client for the contact service
	rpc *rpc.Client[contactv1.ContactsClient]
	tls *infratls.Config
}

func New(logger *slog.Logger, discovery discovery.DiscoveryProvider, tls *infratls.Config) (*Client, error) {
	// [FACTORY] Required by go-kit to instantiate the gRPC stub
	factory := func(conn *grpc.ClientConn) contactv1.ContactsClient {
		return contactv1.NewContactsClient(conn)
	}

	// [INIT] Initialize the shared RPC client wrapper
	c, err := webitel.New(logger, discovery, ServiceName, tls, factory)
	if err != nil {
		return nil, fmt.Errorf("[im-contact-client] initialization failed: %w", err)
	}

	return &Client{
		logger: logger,
		rpc:    c,
	}, nil
}

// SearchContact performs a contact lookup using the resilient RPC execution wrapper
func (c *Client) SearchContact(ctx context.Context, req *contactv1.SearchContactRequest) (*contactv1.ContactList, error) {
	var resp *contactv1.ContactList

	// [EXECUTE] go-kit's Execute handles load balancing, retries, and error mapping
	err := c.rpc.Execute(ctx, func(api contactv1.ContactsClient) error {
		c.logger.Debug("CONTACTS.SEARCH_CONTACT", slog.Any("req", req))

		var err error
		resp, err = api.SearchContact(ctx, req)
		return err
	})

	return resp, err
}

// Close gracefully shuts down the underlying gRPC connection pool
func (c *Client) Close() error {
	if c.rpc != nil {
		return c.rpc.Close()
	}
	return nil
}
