package common

import (
	"context"
	"fmt"

	"github.com/webitel/im-providers-service/gen/go/gateway/v1"
	imgateway "github.com/webitel/im-providers-service/infra/client/grpc/im-gateway"
	"github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/webitel-go-kit/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type CoreMessanger interface {
	SendText(ctx context.Context, in *model.SendTextRequest) (*model.SendTextResponse, error)
	SendImage(ctx context.Context, in *model.SendImageRequest) (*model.SendImageResponse, error)
	SendDocument(ctx context.Context, in *model.SendDocumentRequest) (*model.SendDocumentResponse, error)
	SendContact(ctx context.Context, in *model.SendContactRequest) (*model.SendResponse, error)
	SendLocation(ctx context.Context, in *model.SendLocationRequest) (*model.SendResponse, error)
}

const newContactsSource string = "whatsapp"

const (
	XWebitelTypeHeader     string = "x-webitel-type"
	XWebitelTypeProvider   string = "provider"
	XWebitelProviderHeader string = "x-webitel-provider"
)

type decoratedCoreMessanger struct {
	CoreMessanger

	gatewayClient *imgateway.Client
}

func newDecoratedCoreMessanger(coreMessanger CoreMessanger, gatewayClient *imgateway.Client) *decoratedCoreMessanger {
	return &decoratedCoreMessanger{CoreMessanger: coreMessanger, gatewayClient: gatewayClient}
}

func (decoratedCoreMessanger *decoratedCoreMessanger) SendText(ctx context.Context, in *model.SendTextRequest) (*model.SendTextResponse, error) {
	requestContext, err := decoratedCoreMessanger.prepareOutgoingSendCoreRequest(ctx, in.From, in.To, int(in.DomainID))
	if err != nil {
		return nil, err
	}

	return decoratedCoreMessanger.CoreMessanger.SendText(requestContext, in)
}

func (decoratedCoreMessanger *decoratedCoreMessanger) prepareOutgoingSendCoreRequest(ctx context.Context, from, to model.Peer, dc int) (context.Context, error) {
	if err := decoratedCoreMessanger.resolveInternalContactIdentity(ctx, to, from, dc); err != nil {
		return ctx, err
	}

	outgoingContext, err := decoratedCoreMessanger.prepareOutCallMetadata(ctx, dc, from.Sub)
	if err != nil {
		return ctx, err
	}

	return outgoingContext, nil
}

func (decoratedCoreMessanger *decoratedCoreMessanger) resolveInternalContactIdentity(ctx context.Context, to, from model.Peer, dc int) error {
	outgoingContext, err := decoratedCoreMessanger.prepareOutCallMetadata(ctx, dc, to.Sub)
	if err != nil {
		return err
	}

	createdContact, err := decoratedCoreMessanger.gatewayClient.Create(outgoingContext, &gateway.CreateContactRequest{
		IssId:    from.Iss,
		Type:     newContactsSource,
		Name:     from.Name,
		Username: from.Name, //TODO
		Metadata: map[string]string{},
		Subject:  from.Sub,
		DomainId: int32(dc),
		IsBot:    false,
	})

	if err != nil && status.Code(err) != codes.AlreadyExists {
		return errors.Internal("executing create contact gateway request", errors.WithCause(err), errors.WithID("whatsapp.webhook.core.messanger.resolve_internal_contact_identity"))
	}

	_, err = decoratedCoreMessanger.gatewayClient.CreateVia(
		outgoingContext,
		&gateway.ViasServiceCreateRequest{
			Iss: &createdContact.Iss,
			Sub: &createdContact.Sub,
			Via: to.ID.String(),
		},
	)

	if err != nil {
		errorCode := status.Code(err)
		if errorCode == codes.AlreadyExists {
			return nil
		}

		return errors.New(
			"executing create via gateway request",
			errors.WithCode(errorCode),
			errors.WithCause(err),
			errors.WithID("whatsapp.webhook.core.messanger.resolve_internal_contact_identity"),
			errors.WithValue("iss", from.Iss),
			errors.WithValue("sub", from.Sub),
		)
	}

	return nil
}

func (decoratedCoreMessanger *decoratedCoreMessanger) prepareOutCallMetadata(ctx context.Context, dc int, sub string) (context.Context, error) {
	if dc <= 0 {
		return ctx, errors.InvalidArgument("domain id is required", errors.WithID("whatsapp.core.messanger.prepare_out_call_metadata"))
	}

	if sub == "" {
		return ctx, errors.InvalidArgument("sub is required", errors.WithID("whatsapp.core.messanger.prepare_out_call_metadata"))
	}

	metadataIdentityKey := fmt.Sprintf("%d.%s", dc, sub)

	md := metadata.Pairs(
		XWebitelTypeHeader, XWebitelTypeProvider,
		XWebitelProviderHeader, metadataIdentityKey,
	)

	pairedCtx := metadata.NewOutgoingContext(ctx, md)

	return pairedCtx, nil
}

func (decoratedCoreMessanger *decoratedCoreMessanger) SendImage(ctx context.Context, in *model.SendImageRequest) (*model.SendImageResponse, error) {
	requestContext, err := decoratedCoreMessanger.prepareOutgoingSendCoreRequest(ctx, in.From, in.To, int(in.DomainID))
	if err != nil {
		return nil, err
	}

	return decoratedCoreMessanger.CoreMessanger.SendImage(requestContext, in)
}

func (decoratedCoreMessanger *decoratedCoreMessanger) SendDocument(ctx context.Context, in *model.SendDocumentRequest) (*model.SendDocumentResponse, error) {
	requestContext, err := decoratedCoreMessanger.prepareOutgoingSendCoreRequest(ctx, in.From, in.To, int(in.DomainID))
	if err != nil {
		return nil, err
	}

	return decoratedCoreMessanger.CoreMessanger.SendDocument(requestContext, in)
}

func (decoratedCoreMessanger *decoratedCoreMessanger) SendLocation(ctx context.Context, in *model.SendLocationRequest) (*model.SendResponse, error) {
	requestContext, err := decoratedCoreMessanger.prepareOutgoingSendCoreRequest(ctx, in.From, in.To, in.DomainID)
	if err != nil {
		return nil, err
	}

	return decoratedCoreMessanger.CoreMessanger.SendLocation(requestContext, in)
}

func (decoratedCoreMessanger *decoratedCoreMessanger) SendContact(ctx context.Context, in *model.SendContactRequest) (*model.SendResponse, error) {
	requestContext, err := decoratedCoreMessanger.prepareOutgoingSendCoreRequest(ctx, in.From, in.To, in.DomainID)
	if err != nil {
		return nil, err
	}

	return decoratedCoreMessanger.CoreMessanger.SendContact(requestContext, in)
}
