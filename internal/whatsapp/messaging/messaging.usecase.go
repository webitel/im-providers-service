package messaging

import (
	"context"
	"log/slog"

	"github.com/webitel/im-providers-service/internal/whatsapp/common"
	"github.com/webitel/webitel-go-kit/pkg/errors"
)

type ResolveWhatsAppBusinessAccountQuery struct {
	BotIss string
	BotSub string
}

type WhatsAppBusinessAccountResolver interface {
	Resolve(ctx context.Context, query ResolveWhatsAppBusinessAccountQuery) (*WhatsAppBusinessAccount, error)
}

type messaging struct {
	logger                          *slog.Logger
	encryptor                       common.Encryptor
	whatsAppBusinessAccountResolver WhatsAppBusinessAccountResolver
}

func newMessaging(logger *slog.Logger, encryptor common.Encryptor, whatsAppBusinessAccountResolver WhatsAppBusinessAccountResolver) *messaging {
	return &messaging{logger: logger, encryptor: encryptor, whatsAppBusinessAccountResolver: whatsAppBusinessAccountResolver}
}

func (messaging *messaging) prepareMessageManager(ctx context.Context, from common.Contact) (*MessageManager, error) {
	resolveQuery := ResolveWhatsAppBusinessAccountQuery{BotIss: from.Iss, BotSub: from.Sub}

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

func (messaging *messaging) SendText(ctx context.Context) {}
