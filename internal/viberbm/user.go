package viberbm

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	gatewayv1 "github.com/webitel/im-providers-service/gen/go/gateway/v1"
	grpcclient "github.com/webitel/im-providers-service/infra/client/grpc"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	vibbmmodel "github.com/webitel/im-providers-service/internal/viberbm/model"
)

// syncContact resolves (creating if needed) the internal contact for a Viber BM
// user, caching the result to skip the gateway round-trip on redelivery.
// Idempotent; swallows AlreadyExists.
func (p *viberBMProvider) syncContact(
	ctx context.Context,
	gate *vibbmmodel.ViberBMGate,
	from, displayName string,
) (*gatewayv1.Contact, error) {
	// Gate-scoped key: an MSISDN is global, so without the gate prefix the same
	// customer on a second gate would short-circuit and never get its Via link.
	cacheUser := toExternalUser(gate.ID+":"+from, displayName)
	log := p.logger.With("gate_id", gate.ID, "from", from)

	if known, _ := p.userCache.IsKnown(ctx, cacheUser); known {
		log.DebugContext(ctx, "viber_bm contact: cache hit, skipping gateway sync")

		return &gatewayv1.Contact{Sub: from}, nil
	}

	authCtx := withGatewayIdentity(ctx, gate)

	contact, created, err := p.ensureContact(authCtx, toExternalUser(from, displayName))
	if err != nil {
		log.ErrorContext(ctx, "viber_bm contact: create failed", "err", err)

		return nil, err
	}

	if created {
		log.InfoContext(ctx, "viber_bm contact: created", "contact_iss", contact.GetIss())
	} else {
		log.DebugContext(ctx, "viber_bm contact: already exists")
	}

	p.ensureVia(authCtx, &from, &contact.Iss, gate.ID)

	if err := p.userCache.MarkKnown(ctx, cacheUser); err != nil {
		log.WarnContext(ctx, "viber_bm contact: cache mark failed", "err", err)
	}

	return contact, nil
}

func (p *viberBMProvider) ensureContact(ctx context.Context, user *sharedmodel.ExternalUser) (*gatewayv1.Contact, bool, error) {
	contact, err := p.gatewayer.Create(ctx, &gatewayv1.CreateContactRequest{
		IssId: p.Type(),
		Type:  p.Type(),
		// Username must be unique per issuer, so it is the MSISDN, not the
		// non-unique pushName (which is the display Name).
		Name:     user.FirstName,
		Username: user.ID,
		Subject:  user.ID,
	})
	if err != nil {
		if isAlreadyExists(err) {
			return &gatewayv1.Contact{Sub: user.ID, Iss: p.Type()}, false, nil
		}

		return nil, false, fmt.Errorf("create contact: %w", err)
	}

	return contact, true, nil
}

// ensureVia links the gate to the contact as a "via" channel; errors non-fatal.
func (p *viberBMProvider) ensureVia(ctx context.Context, contactSub, contactIss *string, gateID string) {
	_, err := p.gatewayer.CreateVia(ctx, &gatewayv1.ViasServiceCreateRequest{
		Via: gateID,
		Iss: contactIss,
		Sub: contactSub,
	})
	switch {
	case err == nil:
		p.logger.DebugContext(ctx, "viber_bm via: linked", "gate_id", gateID, "contact_sub", *contactSub)
	case isAlreadyExists(err):
		p.logger.DebugContext(ctx, "viber_bm via: already linked", "gate_id", gateID, "contact_sub", *contactSub)
	default:
		p.logger.WarnContext(ctx, "viber_bm via: create failed", "gate_id", gateID, "contact_sub", *contactSub, "err", err)
	}
}

// toExternalUser maps a Viber BM sender to the domain user; display name falls
// back to the MSISDN (forwards carry no rich profile).
func toExternalUser(from, displayName string) *sharedmodel.ExternalUser {
	name := displayName
	if name == "" {
		name = from
	}

	return &sharedmodel.ExternalUser{
		ID:        from,
		FirstName: name,
	}
}

// withGatewayIdentity attaches the domain-scoped caller identity required by
// the im-gateway service to authenticate inbound gRPC calls.
func withGatewayIdentity(ctx context.Context, gate *vibbmmodel.ViberBMGate) context.Context {
	id := fmt.Sprintf("%d.%s", gate.DomainID, gate.Peer.Sub)

	return grpcclient.WithIdentity(ctx, grpcclient.StringIdentity(id))
}

func isAlreadyExists(err error) bool {
	st, ok := status.FromError(err)

	return ok && st.Code() == codes.AlreadyExists
}
