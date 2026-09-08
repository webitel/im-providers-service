package custom

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	gatewayv1 "github.com/webitel/im-providers-service/gen/go/gateway/v1"
	grpcclient "github.com/webitel/im-providers-service/infra/client/grpc"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	custommodel "github.com/webitel/im-providers-service/internal/custom/model"
)

// contactSub reproduces the identity rule of the previous gateway: an external
// user is "{sender.type}|{sender.id}", so one gate can front several source
// channels without their user ids colliding, and an existing integration keeps
// addressing the same contacts it always did.
func contactSub(s *wireSender) string {
	if s == nil {
		return ""
	}

	kind := strings.TrimSpace(s.Type)
	if kind == "" {
		return s.ID
	}

	return kind + "|" + s.ID
}

func (p *customProvider) syncContact(ctx context.Context, gate *custommodel.CustomGate, s *wireSender) (*gatewayv1.Contact, error) {
	user := toExternalUser(s)

	if known, _ := p.userCache.IsKnown(ctx, user); known {
		return &gatewayv1.Contact{Sub: user.ID, Iss: p.Type()}, nil
	}

	authCtx := withGatewayIdentity(ctx, gate)

	contact, err := p.ensureContact(authCtx, user)
	if err != nil {
		return nil, err
	}

	p.ensureVia(authCtx, &user.ID, &contact.Iss, gate.ID)
	_ = p.userCache.MarkKnown(ctx, user)

	return contact, nil
}

func (p *customProvider) ensureContact(ctx context.Context, user *sharedmodel.ExternalUser) (*gatewayv1.Contact, error) {
	name := user.FirstName
	if name == "" {
		name = user.ID
	}

	contact, err := p.gatewayer.Create(ctx, &gatewayv1.CreateContactRequest{
		IssId:    p.Type(),
		Type:     p.Type(),
		Name:     name,
		Username: name,
		Subject:  user.ID,
	})
	if err != nil {
		if isAlreadyExists(err) {
			return &gatewayv1.Contact{Sub: user.ID, Iss: p.Type()}, nil
		}

		return nil, fmt.Errorf("create contact: %w", err)
	}

	return contact, nil
}

func (p *customProvider) ensureVia(ctx context.Context, contactSub, contactIss *string, gateID string) {
	_, err := p.gatewayer.CreateVia(ctx, &gatewayv1.ViasServiceCreateRequest{
		Via: gateID,
		Iss: contactIss,
		Sub: contactSub,
	})
	if err != nil && !isAlreadyExists(err) {
		p.logger.Warn("create via: skipped", "contact", contactSub, "gate_id", gateID, "err", err)
	}
}

func toExternalUser(s *wireSender) *sharedmodel.ExternalUser {
	name := ""
	if s != nil {
		name = s.Name
		if name == "" {
			name = s.Nickname
		}
	}

	return &sharedmodel.ExternalUser{
		ID:        contactSub(s),
		FirstName: name,
	}
}

func withGatewayIdentity(ctx context.Context, gate *custommodel.CustomGate) context.Context {
	id := fmt.Sprintf("%d.%s", gate.DomainID, gate.Peer.Sub)

	return grpcclient.WithIdentity(ctx, grpcclient.StringIdentity(id))
}

func isAlreadyExists(err error) bool {
	st, ok := status.FromError(err)

	return ok && st.Code() == codes.AlreadyExists
}
