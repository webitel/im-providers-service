package facebook

import (
	"context"
	"fmt"

	gatewayv1 "github.com/webitel/im-providers-service/gen/go/gateway/v1"
	grpcclient "github.com/webitel/im-providers-service/infra/client/grpc"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	fbmodel "github.com/webitel/im-providers-service/internal/facebook/model"
	"github.com/webitel/webitel-go-kit/pkg/semconv"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// syncContact resolves the internal contact for a Facebook user, creating it
// if necessary. The result is cached so repeated webhook deliveries from the
// same PSID skip the gateway round-trip.
func (p *facebookProvider) syncContact(
	ctx context.Context,
	gate *fbmodel.FacebookGate,
	psid string,
	profile *UserProfile,
) (*gatewayv1.Contact, error) {
	user := toExternalUser(psid, profile)

	if known, _ := p.userCache.IsKnown(ctx, user); known {
		return &gatewayv1.Contact{Sub: psid}, nil
	}

	authCtx := withGatewayIdentity(ctx, gate)

	contact, err := p.ensureContact(authCtx, user)
	if err != nil {
		return nil, err
	}

	p.ensureVia(authCtx, &psid, &contact.Iss, gate.ID)
	_ = p.userCache.MarkKnown(ctx, user)

	return contact, nil
}

// ensureContact creates the internal contact or returns a stub when the
// contact already exists (idempotent by design on the gateway side).
func (p *facebookProvider) ensureContact(ctx context.Context, user *sharedmodel.ExternalUser) (*gatewayv1.Contact, error) {
	contact, err := p.gatewayer.Create(ctx, &gatewayv1.CreateContactRequest{
		IssId:    p.Type(),
		Type:     p.Type(),
		Name:     user.FirstName,
		Username: user.LastName,
		Subject:  user.ID,
	})
	if err != nil {
		if isAlreadyExists(err) {
			return &gatewayv1.Contact{Sub: user.ID}, nil
		}
		return nil, fmt.Errorf("create contact: %w", err)
	}
	return contact, nil
}

// ensureVia links the gate to the internal contact as a "via" channel.
// Errors are non-fatal — AlreadyExists is silently ignored.
func (p *facebookProvider) ensureVia(ctx context.Context, contactSub, contactIss *string, gateID string) {
	via, err := p.gatewayer.CreateVia(ctx, &gatewayv1.ViasServiceCreateRequest{
		Via: gateID,
		Iss: contactIss,
		Sub: contactSub,
	})
	p.logger.Debug("create via: done", "contact", contactSub, "gate_id", gateID, "via", via)

	if err != nil && !isAlreadyExists(err) {
		p.logger.Warn("create via: skipped", "contact", contactSub, "gate_id", gateID, semconv.ErrorKey, err)
	}
}

// toExternalUser maps a Facebook user profile to the domain cache key.
func toExternalUser(psid string, profile *UserProfile) *sharedmodel.ExternalUser {
	return &sharedmodel.ExternalUser{
		ID:        psid,
		FirstName: profile.FirstName,
		LastName:  profile.LastName,
	}
}

// withGatewayIdentity attaches the domain-scoped caller identity required by
// the im-gateway service to authenticate inbound gRPC calls.
func withGatewayIdentity(ctx context.Context, gate *fbmodel.FacebookGate) context.Context {
	id := fmt.Sprintf("%d.%s", gate.DomainID, gate.Peer.Sub)
	return grpcclient.WithIdentity(ctx, grpcclient.StringIdentity(id))
}

// isAlreadyExists reports whether a gRPC error carries the AlreadyExists code.
func isAlreadyExists(err error) bool {
	st, ok := status.FromError(err)
	return ok && st.Code() == codes.AlreadyExists
}
