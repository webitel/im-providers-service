package messaging

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/webitel/im-providers-service/gen/go/gateway/v1"
	imgateway "github.com/webitel/im-providers-service/infra/client/grpc/im-gateway"
	"github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/im-providers-service/internal/whatsapp/common"
	"github.com/webitel/im-providers-service/internal/whatsapp/messaging/components"
	"github.com/webitel/webitel-go-kit/pkg/errors"
	"google.golang.org/grpc/metadata"
)

type ResolveWhatsAppBusinessAccountQuery struct {
	BotIss string
	BotSub string
	GateID *string
}

type WhatsAppBusinessAccountResolver interface {
	Resolve(ctx context.Context, query ResolveWhatsAppBusinessAccountQuery) (*WhatsAppBusinessAccount, error)
}

type Messaging struct {
	logger                          *slog.Logger
	encryptor                       common.Encryptor
	whatsAppBusinessAccountResolver WhatsAppBusinessAccountResolver
	gatewayClient                   *imgateway.Client
}

func newMessaging(logger *slog.Logger, encryptor common.Encryptor, whatsAppBusinessAccountResolver WhatsAppBusinessAccountResolver, gatewayClient *imgateway.Client) *Messaging {
	return &Messaging{logger: logger, encryptor: encryptor, whatsAppBusinessAccountResolver: whatsAppBusinessAccountResolver, gatewayClient: gatewayClient}
}

func (messaging *Messaging) prepareMessageManagerFromBusinessAccount(businessAccount *WhatsAppBusinessAccount) (*MessageManager, error) {
	preparedWhatsAppBusinessAccount, err := businessAccount.PostFetch(messaging.encryptor)
	if err != nil {
		return nil, err
	}

	if preparedWhatsAppBusinessAccount.IsTokenExpired() {
		return nil, errors.Unauthenticated(
			"access token is expired for this whatsapp business account",
			errors.WithID("messaging.usecase.prepare_message_manager"),
			errors.WithValue("phone_number_id", preparedWhatsAppBusinessAccount.PhoneNumberID),
			errors.WithValue("access_token_expires_at", preparedWhatsAppBusinessAccount.AccessTokenExpiresAt),
		)
	}

	whatsAppBusinessAccountRequestClient, err := preparedWhatsAppBusinessAccount.CreateRequestClient()
	if err != nil {
		return nil, err
	}

	whatsAppBusinessAccountMessageManager := newMessageManager(*whatsAppBusinessAccountRequestClient, preparedWhatsAppBusinessAccount.PhoneNumberID)

	return whatsAppBusinessAccountMessageManager, nil
}

func (messaging *Messaging) prepareMessageManager(ctx context.Context, from common.Contact, gateID *string) (*MessageManager, error) {
	resolveQuery := ResolveWhatsAppBusinessAccountQuery{BotIss: from.Iss, BotSub: from.Sub, GateID: gateID}

	whatsAppBusinessAccount, err := messaging.whatsAppBusinessAccountResolver.Resolve(ctx, resolveQuery)
	if err != nil {
		return nil, err
	}

	preparedWhatsAppBusinessAccount, err := whatsAppBusinessAccount.PostFetch(messaging.encryptor)
	if err != nil {
		return nil, err
	}

	if preparedWhatsAppBusinessAccount.IsTokenExpired() {
		return nil, errors.Unauthenticated(
			"access token is expired for this whatsapp business account",
			errors.WithID("messaging.usecase.prepare_message_manager"),
			errors.WithValue("phone_number_id", preparedWhatsAppBusinessAccount.PhoneNumberID),
			errors.WithValue("access_token_expires_at", preparedWhatsAppBusinessAccount.AccessTokenExpiresAt),
		)
	}

	whatsAppBusinessAccountRequestClient, err := preparedWhatsAppBusinessAccount.CreateRequestClient()
	if err != nil {
		return nil, err
	}

	whatsAppBusinessAccountMessageManager := newMessageManager(*whatsAppBusinessAccountRequestClient, preparedWhatsAppBusinessAccount.PhoneNumberID)

	return whatsAppBusinessAccountMessageManager, nil
}

func extractGateID(gate string) *string {
	if gate != "" {
		return &gate
	}

	return nil
}

type outboundMessagePreparationInfo struct {
	messageManager      *MessageManager
	externalPhoneNumber string
}

func (messaging *Messaging) prepareOutboundMessageInfo(ctx context.Context, message *model.Message) (*outboundMessagePreparationInfo, error) {
	businessAccount, err := messaging.whatsAppBusinessAccountResolver.Resolve(ctx, ResolveWhatsAppBusinessAccountQuery{GateID: extractGateID(message.GateID)})
	if err != nil {
		return nil, err
	}

	whatsAppManager, err := messaging.prepareMessageManagerFromBusinessAccount(businessAccount)
	if err != nil {
		return nil, err
	}

	metadataIdentityKey := fmt.Sprintf("%d.%s", message.DomainID, businessAccount.Contact.Sub)
	md := metadata.Pairs(
		"x-webitel-type", "provider",
		"x-webitel-provider", metadataIdentityKey,
	)

	contact, err := messaging.gatewayClient.Locate(
		metadata.NewOutgoingContext(ctx, md),
		&gateway.LocateConatctRequest{
			Id:       message.To.Sub,
			DomainId: message.DomainID,
		},
	)

	if err != nil {
		return nil, err
	}

	return &outboundMessagePreparationInfo{
		messageManager:      whatsAppManager,
		externalPhoneNumber: contact.GetItem().GetSub(),
	}, nil
}

func (messaging *Messaging) SendText(ctx context.Context, req *model.Message) (*model.MessageResponse, error) {
	sendingInfo, err := messaging.prepareOutboundMessageInfo(ctx, req)
	if err != nil {
		return nil, err
	}

	textMessage, err := components.NewTextMessage(components.TextMessageConfigs{
		Text:         req.Text,
		AllowPreview: false,
	})

	if err != nil {
		return nil, errors.Wrap(err, errors.WithID("messaging.usecase.send_text"))
	}

	response, err := sendingInfo.messageManager.Send(ctx, textMessage, sendingInfo.externalPhoneNumber)
	if err != nil {
		return nil, err
	}

	if response.Error != nil {
		return nil, errors.New("sending whatsapp text message", errors.WithCause(response.Error.ToGRPCError()), errors.WithID("messaging.usecase.send_text"))
	}

	sendMessageID := ""
	if len(response.Messages) > 0 {
		sendMessageID = response.Messages[0].ID
	}

	return &model.MessageResponse{ID: sendMessageID}, nil
}

func (messaging *Messaging) SendImage(ctx context.Context, req *model.Message) (*model.MessageResponse, error) {
	sendingInfo, err := messaging.prepareOutboundMessageInfo(ctx, req)
	if err != nil {
		return nil, err
	}

	imageMessage, err := components.NewImageMessage(components.ImageMessageConfigs{
		Link:    req.Images[0].URL,
		Caption: &req.Text,
	})

	if err != nil {
		return nil, err
	}

	response, err := sendingInfo.messageManager.Send(ctx, imageMessage, sendingInfo.externalPhoneNumber)
	if err != nil {
		return nil, err
	}

	if response.Error != nil {
		return nil, errors.New("sending whatsapp image message", errors.WithCause(response.Error.ToGRPCError()), errors.WithID("messaging.usecase.send_text"))
	}

	sendMessageID := ""
	if len(response.Messages) > 0 {
		sendMessageID = response.Messages[0].ID
	}

	return &model.MessageResponse{ID: sendMessageID}, nil
}

func (messaging *Messaging) SendDocument(ctx context.Context, req *model.Message) (*model.MessageResponse, error) {
	sendingInfo, err := messaging.prepareOutboundMessageInfo(ctx, req)
	if err != nil {
		return nil, err
	}

	documentMessage, err := components.NewDocumentMessage(components.DocumentMessageConfigs{
		Link:     req.Documents[0].URL,
		Caption:  &req.Text,
		FileName: req.Documents[0].FileName,
	})

	if err != nil {
		return nil, err
	}

	response, err := sendingInfo.messageManager.Send(ctx, documentMessage, sendingInfo.externalPhoneNumber)
	if err != nil {
		return nil, err
	}

	if response.Error != nil {
		return nil, errors.New("sending whatsapp image message", errors.WithCause(response.Error.ToGRPCError()), errors.WithID("messaging.usecase.send_text"))
	}

	sendMessageID := ""
	if len(response.Messages) > 0 {
		sendMessageID = response.Messages[0].ID
	}

	return &model.MessageResponse{ID: sendMessageID}, nil
}
