package standard

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	"go.uber.org/fx"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	authv1pb "github.com/webitel/im-providers-service/gen/go/auth/v1"
	contactv1pb "github.com/webitel/im-providers-service/gen/go/contact/v1"
	interfaces "github.com/webitel/im-providers-service/infra/auth"
	authclient "github.com/webitel/im-providers-service/infra/client/grpc/im-auth"
	contactclient "github.com/webitel/im-providers-service/infra/client/grpc/im-contact"
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
	logger    *slog.Logger
	auther    *authclient.Client
	contacter *contactclient.Client
}

func New(logger *slog.Logger, auther *authclient.Client, contacter *contactclient.Client) (*Authorizer, error) {
	if auther == nil {
		return nil, errors.New("no auth client provided")
	}
	if contacter == nil {
		return nil, errors.New("no contact client provided")
	}
	return &Authorizer{
		logger:    logger,
		auther:    auther,
		contacter: contacter,
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

func (da *Authorizer) resolveServiceIdentity(ctx context.Context) (*Identity, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.Forbidden("metadata required for internal identity resolve")
	}
	authType := getHeader(md, interfaces.XWebitelTypeHeader)
	switch authType {
	case string(interfaces.XWebitelTypeSchema):
		return da.resolveSchemaIdentity(ctx, md)
	case string(interfaces.XWebitelTypeEngine):
		return da.resolveUserIdentity(ctx)
	default:
		return nil, errors.Forbidden("unsupported auth type")
	}
}

func (da *Authorizer) resolveSchemaIdentity(ctx context.Context, md metadata.MD) (*Identity, error) {
	rawSchema := getHeader(md, interfaces.SchemaIdentificationHeader)
	if rawSchema == "" {
		return nil, errors.Forbidden("special header required")
	}

	domainID, sub, err := splitDomainAndSub(rawSchema)
	if err != nil {
		return nil, err
	}

	if domainID == 0 || sub == "" {
		return nil, errors.Forbidden("special header format: {domain_id}.{flow_id} required")
	}

	res, err := da.contacter.SearchContact(ctx, &contactv1pb.SearchContactRequest{
		Subjects: []string{sub},
		DomainId: int32(domainID),
		Size:     1,
	})

	if err != nil || len(res.GetContacts()) == 0 {
		return nil, errors.NotFound("bot contact not found")
	}

	return &Identity{
		ContactID: res.GetContacts()[0].GetId(),
		DomainID:  domainID,
		Name:      coalesce(res.GetContacts()[0].GetName(), res.GetContacts()[0].GetUsername()),
	}, nil
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

// --- Internal Helpers ---

func splitDomainAndSub(raw string) (int64, string, error) {
	if raw == "" {
		return 0, "", errors.New("empty domain")
	}

	parts := strings.SplitN(raw, ".", 2)
	domainID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, "", err
	}

	if len(parts) < 2 {
		return domainID, "", nil
	}
	return domainID, parts[1], nil
}

func getHeader(md metadata.MD, key string) string {
	if vals := md.Get(key); len(vals) > 0 {
		return vals[0]
	}

	return ""
}
