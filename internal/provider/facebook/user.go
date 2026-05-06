package facebook

import (
	"context"
	"fmt"

	gatewayv1 "github.com/webitel/im-providers-service/gen/go/gateway/v1"
	client "github.com/webitel/im-providers-service/infra/client/grpc"
	"github.com/webitel/im-providers-service/internal/domain/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (p *facebookProvider) externalUserSync(ctx context.Context, g *model.FacebookGate, psid string, prof *UserProfile) (*gatewayv1.Contact, error) {
	u := &model.ExternalUser{ID: psid, FirstName: prof.FirstName, LastName: prof.LastName}

	if ok, _ := p.userCache.IsKnown(ctx, u); ok {
		return &gatewayv1.Contact{Sub: psid}, nil
	}

	authCtx := client.WithIdentity(ctx, client.StringIdentity(fmt.Sprintf("%d.%s", g.DomainID, g.Peer.Sub)))

	internalUsr, err := p.gatewayer.Create(
		authCtx,
		&gatewayv1.CreateContactRequest{
			IssId:    g.Peer.Iss,
			Type:     p.Type(),
			Name:     u.FirstName,
			Username: u.LastName,
			Subject:  u.ID,
			IsBot:    false,
		})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.AlreadyExists {
			_ = p.userCache.MarkKnown(ctx, u)
			return &gatewayv1.Contact{Sub: psid}, nil
		}
		return nil, fmt.Errorf("gateway sync failed: %w", err)
	}

	_ = p.userCache.MarkKnown(ctx, u)
	return internalUsr, nil
}
