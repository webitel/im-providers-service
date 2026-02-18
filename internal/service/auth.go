package service

import (
	"context"
	"fmt"

	gatewayv1 "github.com/webitel/im-providers-service/gen/go/gateway/v1"
	imauth "github.com/webitel/im-providers-service/infra/client/grpc/im-gateway"
	"github.com/webitel/im-providers-service/internal/domain/model"
	"google.golang.org/grpc/metadata"
)

var _ Auther = (*AuthService)(nil)

// Auther defines the behavior for session validation.
type Auther interface {
	Inspect(ctx context.Context) (*model.AuthContact, error)
}

type AuthService struct {
	client *imauth.Client
}

func NewAuthService(client *imauth.Client) *AuthService {
	return &AuthService{client: client}
}

// Inspect transparently redirects all incoming metadata to the auth service.
func (s *AuthService) Inspect(ctx context.Context) (*model.AuthContact, error) {
	// [METADATA_EXTRACTION] Capture all incoming headers
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("no metadata found in context")
	}

	// [FULL_REDIRECT] Pass all original headers to the outgoing call
	// This includes X-Webitel-Access, X-Webitel-Device, X-Webitel-Client, and others.
	outCtx := metadata.NewOutgoingContext(ctx, md)

	// [IDENTITY_INSPECTION]
	// We send an empty request body because the token is already in the metadata headers.
	auth, err := s.client.Inspect(outCtx, &gatewayv1.InspectRequest{})
	if err != nil {
		return nil, fmt.Errorf("identity inspection failed: %w", err)
	}

	return &model.AuthContact{
		ContactID: auth.Contact.Id,
		Sub:       auth.Contact.Sub,
		Iss:       auth.Contact.Iss,
		Name:      auth.Contact.Name,
	}, nil
}
