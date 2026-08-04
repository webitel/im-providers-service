package bot

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/webitel/im-providers-service/gen/go/contact/v1"
	gatewayv1 "github.com/webitel/im-providers-service/gen/go/gateway/v1"
	grpcclient "github.com/webitel/im-providers-service/infra/client/grpc"
	coremodel "github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/im-providers-service/internal/telegram/bot/model"
	"github.com/webitel/webitel-go-kit/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// viaHeader identifies which gate/via a message forwarded to im-gateway-service
// belongs to, so the same external contact sub can be disambiguated across
// multiple Telegram bots. Must match the Via value registered via ensureVia.
const viaHeader = "x-webitel-via"

func (p *Provider) constructFrom(
	ctx context.Context,
	gate *model.Gate,
	user *model.User,
) (*coremodel.Peer, error) {

	via := gate.ID.String()

	externalUser := p.toExternalUser(user)
	res := &coremodel.Peer{
		Sub:  externalUser.ID,
		Iss:  p.Type(),
		Type: coremodel.PeerUser,
		Name: externalUser.FirstName,
		Via:  &via,
	}

	// if known, _ := p.userCache.IsKnown(ctx, externalUser); known {
	// return res, nil
	// }

	authCtx := withGatewayIdentity(ctx, gate)

	contact, err := p.ensureContact(authCtx, externalUser)
	if err != nil {
		return nil, err
	}

	err = p.ensureVia(authCtx, &externalUser.ID, &contact.Iss, gate.ID)
	if err != nil {
		return nil, err
	}

	// _ = p.userCache.MarkKnown(ctx, externalUser)

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

// withVia attaches the gate's via identifier to the outgoing gRPC metadata
// for calls to im-gateway-service.
func withVia(ctx context.Context, gate *model.Gate) context.Context {
	return metadata.AppendToOutgoingContext(ctx, viaHeader, gate.ID.String())
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

func (p *Provider) fetchContactTelegramID(ctx context.Context, gate *model.Gate, contactID uuid.UUID) (int64, error) {
	if contactID == uuid.Nil {
		return 0, nil
	}
	authCtx := withGatewayIdentity(ctx, gate)
	resp, err := p.contactClient.SearchContact(authCtx, &contact.SearchContactRequest{
		Ids: []string{contactID.String()},
	})
	if err != nil {
		return 0, fmt.Errorf("can't resolve contact %s: %w", contactID, err)
	}
	items := resp.GetContacts()
	if len(items) == 0 || items[0].GetSubject() == "" {
		return 0, nil
	}
	telegramID, err := strconv.ParseInt(items[0].GetSubject(), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid telegram id: %w", err)
	}
	return telegramID, nil
}
