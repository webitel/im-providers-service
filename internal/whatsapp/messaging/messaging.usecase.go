package messaging

import (
	"context"
	"log/slog"

	"github.com/webitel/im-providers-service/gen/go/gateway/v1"
	imgateway "github.com/webitel/im-providers-service/infra/client/grpc/im-gateway"
	"github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/im-providers-service/internal/whatsapp/common"
	"github.com/webitel/im-providers-service/internal/whatsapp/messaging/components"
	"github.com/webitel/webitel-go-kit/pkg/errors"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/status"
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
	return &Messaging{logger: logger, encryptor: encryptor, whatsAppBusinessAccountResolver: whatsAppBusinessAccountResolver}
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
	var (
		externalPhoneNmber string
		messageManager     *MessageManager
	)

	egroup, ectx := errgroup.WithContext(ctx)

	egroup.Go(func() error {
		var err error
		messageManager, err = messaging.prepareMessageManager(ectx, common.Contact{}, extractGateID(message.GateID))
		return err
	})

	egroup.Go(func() error {
		contact, err := messaging.gatewayClient.Locate(
			ectx,
			&gateway.LocateConatctRequest{
				Id:       message.To.Sub,
				DomainId: message.DomainID,
			},
		)

		if err != nil {
			if st, ok := status.FromError(err); ok {
				return errors.New("locating external contact information", errors.WithCause(err), errors.WithCode(st.Code()), errors.WithID("messaging.usecase.prepare_message_info"))
			}

			return errors.Internal("locating external contact information", errors.WithCause(err), errors.WithID("messaging.usecase.prepare_message_info"))
		}

		externalPhoneNmber = contact.GetItem().GetSub()

		return nil
	})

	if err := egroup.Wait(); err != nil {
		return nil, err
	}

	return &outboundMessagePreparationInfo{
		messageManager:      messageManager,
		externalPhoneNumber: externalPhoneNmber,
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
	return nil, errors.New("unimplemented", errors.WithID("messaging.usecase.send_image"))
}

func (messaging *Messaging) SendDocument(ctx context.Context, req *model.Message) (*model.MessageResponse, error) {
	return nil, errors.New("unimplemented", errors.WithID("messaging.usecase.send_document"))
}
