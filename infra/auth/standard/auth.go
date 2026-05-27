package standard

import (
	"context"
	"log/slog"

	"go.uber.org/fx"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	authv1pb "github.com/webitel/im-providers-service/gen/go/auth/v1"
	interfaces "github.com/webitel/im-providers-service/infra/auth"
	authclient "github.com/webitel/im-providers-service/infra/client/grpc/im-auth"
)

var Module = fx.Module(
	"default_auth",

	fx.Provide(
		fx.Annotate(
			New,
			fx.As(new(interfaces.Authorizer)),
		),
	),
)

// INTERFACE GUARD
var _ interfaces.Identifier = (*Identity)(nil)

type Identity struct {
	ContactID string
	DomainID  int64
	Name      string
}

func (i *Identity) GetContactID() string {
	return i.ContactID
}

func (i *Identity) GetDomainID() int64 {
	return i.DomainID
}

func (i *Identity) GetName() string {
	return i.Name
}

type Authorizer struct {
	logger *slog.Logger
	auther *authclient.Client
}

func New(logger *slog.Logger, auther *authclient.Client) (*Authorizer, error) {
	if auther == nil {
		return nil, errors.New("no auth client provided")
	}
	return &Authorizer{
		logger: logger,
		auther: auther,
	}, nil
}

// SetIdentity resolves and sets the identity into the derived context.
func (da *Authorizer) SetIdentity(ctx context.Context) (context.Context, error) {
	resolvedIdentity, err := da.resolveIdentity(ctx)
	if err != nil {
		return ctx, errors.Unauthenticated(err.Error())
	}

	newCtx := context.WithValue(ctx, interfaces.AuthContextKey, resolvedIdentity)

	return newCtx, nil
}

// resolveIdentity determines identification path based on connection type and headers
func (da *Authorizer) resolveIdentity(ctx context.Context) (*Identity, error) {
	if client, ok := peer.FromContext(ctx); ok && client.AuthInfo != nil {
		if tlsInfo, ok := client.AuthInfo.(credentials.TLSInfo); ok && len(tlsInfo.State.PeerCertificates) > 0 {
			return da.resolveServiceIdentity(ctx)
		}
	}

	return da.resolveUserIdentity(ctx)
}

func (da *Authorizer) resolveServiceIdentity(_ context.Context) (*Identity, error) {
	return &Identity{}, nil
}

func (da *Authorizer) resolveUserIdentity(ctx context.Context) (*Identity, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.Forbidden("metadata required for user identity resolve")
	}

	auth, err := da.auther.Inspect(metadata.NewOutgoingContext(ctx, md), &authv1pb.InspectRequest{})
	if err != nil {
		return nil, err
	}

	contact := auth.Contact
	if contact == nil {
		return nil, errors.Forbidden("no contact info in authorization")
	}
	return &Identity{
		ContactID: contact.Id,
		DomainID:  auth.Dc,
		Name:      coalesce(contact.Name, contact.GivenName, contact.Username),
	}, nil
}

func coalesce(str ...string) string {
	for _, s := range str {
		if s != "" {
			return s
		}
	}
	return "Unknown"
}
