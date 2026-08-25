package instagram

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	gatewayv1 "github.com/webitel/im-providers-service/gen/go/gateway/v1"
	grpcclient "github.com/webitel/im-providers-service/infra/client/grpc"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	igmodel "github.com/webitel/im-providers-service/internal/instagram/model"
)

// syncContact resolves the internal contact for an Instagram user, creating it
// if necessary. The result is cached so repeated webhook deliveries from the
// same IGSID skip the gateway round-trip.
// The bool return reports whether this call newly created the contact (first
// contact) — used to fire the conversation_started template exactly once.
func (p *instagramProvider) syncContact(
	ctx context.Context,
	gate *igmodel.InstagramGate,
	igsid string,
	profile *UserProfile,
) (*gatewayv1.Contact, bool, error) {
	user := toExternalUser(igsid, profile)

	if known, _ := p.userCache.IsKnown(ctx, user); known {
		return &gatewayv1.Contact{Sub: igsid}, false, nil
	}

	authCtx := withGatewayIdentity(ctx, gate)

	contact, created, err := p.ensureContact(authCtx, user)
	if err != nil {
		return nil, false, err
	}

	p.ensureVia(authCtx, &igsid, &contact.Iss, gate.ID)
	_ = p.userCache.MarkKnown(ctx, user)

	return contact, created, nil
}

// ensureContact creates the internal contact or returns a stub when the
// contact already exists (idempotent by design on the gateway side).
// The bool return is true only when a contact was actually created.
func (p *instagramProvider) ensureContact(ctx context.Context, user *sharedmodel.ExternalUser) (*gatewayv1.Contact, bool, error) {
	contact, err := p.gatewayer.Create(ctx, &gatewayv1.CreateContactRequest{
		IssId:    p.Type(),
		Type:     p.Type(),
		Name:     user.FirstName,
		Username: user.LastName,
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

// ensureVia links the gate to the internal contact as a "via" channel.
// Errors are non-fatal — AlreadyExists is silently ignored.
func (p *instagramProvider) ensureVia(ctx context.Context, contactSub, contactIss *string, gateID string) {
	via, err := p.gatewayer.CreateVia(ctx, &gatewayv1.ViasServiceCreateRequest{
		Via: gateID,
		Iss: contactIss,
		Sub: contactSub,
	})
	p.logger.Debug("create via: done", "contact", contactSub, "gate_id", gateID, "via", via)

	if err != nil && !isAlreadyExists(err) {
		p.logger.Warn("create via: skipped", "contact", contactSub, "gate_id", gateID, "err", err)
	}
}

// toExternalUser maps an Instagram user profile to the domain cache key.
func toExternalUser(igsid string, profile *UserProfile) *sharedmodel.ExternalUser {
	// im-contact requires a non-empty username. Instagram's user node exposes a
	// handle via `username`; fall back to the display name and finally the IGSID
	// so contact creation never fails on an empty value.
	username := profile.Username
	if username == "" {
		username = profile.Name
	}

	if username == "" {
		username = igsid
	}

	name := profile.Name
	if name == "" {
		name = username
	}

	return &sharedmodel.ExternalUser{
		ID:        igsid,
		FirstName: name,
		LastName:  username,
	}
}

// withGatewayIdentity attaches the domain-scoped caller identity required by
// the im-gateway service to authenticate inbound gRPC calls.
func withGatewayIdentity(ctx context.Context, gate *igmodel.InstagramGate) context.Context {
	id := fmt.Sprintf("%d.%s", gate.DomainID, gate.Peer.Sub)

	return grpcclient.WithIdentity(ctx, grpcclient.StringIdentity(id))
}

// isAlreadyExists reports whether a gRPC error carries the AlreadyExists code.
func isAlreadyExists(err error) bool {
	st, ok := status.FromError(err)

	return ok && st.Code() == codes.AlreadyExists
}
