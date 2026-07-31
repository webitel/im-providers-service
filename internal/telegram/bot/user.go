package bot

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	gatewayv1 "github.com/webitel/im-providers-service/gen/go/gateway/v1"
	grpcclient "github.com/webitel/im-providers-service/infra/client/grpc"
	coremodel "github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/im-providers-service/internal/telegram/bot/model"
	"github.com/webitel/webitel-go-kit/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (p *Provider) constructFrom(
	ctx context.Context,
	gate *model.Gate,
	user *model.User,
) (*coremodel.Peer, error) {

	externalUser := p.toExternalUser(user)
	res := &coremodel.Peer{
		Sub:  externalUser.ID,
		Iss:  p.Type(),
		Type: coremodel.PeerUser,
		Name: externalUser.FirstName,
	}

	if known, _ := p.userCache.IsKnown(ctx, externalUser); known {
		return res, nil
	}

	authCtx := withGatewayIdentity(ctx, gate)

	contact, err := p.ensureContact(authCtx, externalUser)
	if err != nil {
		return nil, err
	}

	err = p.ensureVia(authCtx, &contact.Sub, &contact.Iss, gate.ID)
	if err != nil {
		return nil, err
	}

	_ = p.userCache.MarkKnown(ctx, externalUser)

	return res, nil
}

func (p *Provider) toExternalUser(user *model.User) *coremodel.ExternalUser {
	res := &coremodel.ExternalUser{
		ID:        strconv.FormatInt(user.ID, 10),
		FirstName: user.FirstName,
	}

	if user.LastName != nil {
		res.LastName = *user.LastName
	}

	return res
}

func withGatewayIdentity(ctx context.Context, gate *model.Gate) context.Context {
	id := fmt.Sprintf("%d.%s", gate.DC, gate.Bot.Sub)
	return grpcclient.WithIdentity(ctx, grpcclient.StringIdentity(id))
}

func (p *Provider) ensureContact(ctx context.Context, user *coremodel.ExternalUser) (*gatewayv1.Contact, error) {
	contact, err := p.gatewayClient.Create(ctx, &gatewayv1.CreateContactRequest{
		IssId:    p.Type(),
		Type:     p.Type(),
		Name:     user.FirstName,
		Username: user.LastName,
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

func (p *Provider) ensureVia(ctx context.Context, contactSub, contactIss *string, gateID uuid.UUID) error {
	via, err := p.gatewayClient.CreateVia(ctx, &gatewayv1.ViasServiceCreateRequest{
		Via: gateID.String(),
		Iss: contactIss,
		Sub: contactSub,
	})
	p.log.Debug("create via: done", "contact", contactSub, "gate_id", gateID, "via", via)

	if err != nil {
		if !isAlreadyExists(err) {
			return err
		}
	}
	return nil
}

func isAlreadyExists(err error) bool {
	st, ok := status.FromError(err)
	return ok && st.Code() == codes.AlreadyExists
}

func (p *Provider) constructTo(
	ctx context.Context,
	gate *model.Gate,
) (*coremodel.Peer, error) {
	if gate == nil {
		return nil, errors.Internal("gate required to construct to peer")
	}
	if gate.Bot == nil {
		return nil, errors.Internal("gate bot required to construct to peer")
	}
	via := gate.ID.String()

	res := &coremodel.Peer{
		Sub:  gate.Bot.Sub,
		Iss:  gate.Bot.Iss,
		Type: coremodel.PeerUser,
		Name: gate.Bot.Name,
		Via:  &via,
	}
	return res, nil
}
