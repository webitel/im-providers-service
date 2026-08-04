package bot

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	imgateway "github.com/webitel/im-providers-service/infra/client/grpc/im-gateway"
	coremodel "github.com/webitel/im-providers-service/internal/core/model"
	sharedsvc "github.com/webitel/im-providers-service/internal/core/service"
	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
	"github.com/webitel/im-providers-service/internal/provider"
	tgclient "github.com/webitel/im-providers-service/internal/telegram/bot/client"
	"github.com/webitel/im-providers-service/internal/telegram/bot/model"
	"github.com/webitel/im-providers-service/internal/telegram/bot/store"
)

var (
	_ provider.Provider           = (*Provider)(nil)
	_ provider.SignatureValidator = (*Provider)(nil)
)

var noUpdateIDErr = errors.New("update has no ID")

const (
	providerType      = model.ProviderType
	secretTokenHeader = "X-Telegram-Bot-Api-Secret-Token"
)

type IdempotencyCache interface {
	IsProcessed(gateID uuid.UUID, updateID int64) bool
	MarkProcessed(gateID uuid.UUID, updateID int64)
}

func New(
	l *slog.Logger,
	uc sharedstore.ExternalUserCache,
	media sharedsvc.MediaManager,
	store store.TelegramBotStore,
	coreMessageClient sharedsvc.Messenger,
	gatewayer *imgateway.Client,
) *Provider {
	idempotencyCache, err := newIdempotencyCache()
	if err != nil {
		return nil
	}

	return &Provider{
		store:             store,
		log:               l,
		userCache:         uc,
		idempotencyCache:  idempotencyCache,
		coreMessageClient: coreMessageClient,
		tgMessageClient:   tgclient.NewClient(),
		gatewayClient:     gatewayer,
	}
}

type Provider struct {
	store store.TelegramBotStore
	log   *slog.Logger

	userCache        sharedstore.ExternalUserCache
	idempotencyCache IdempotencyCache

	coreMessageClient sharedsvc.Messenger
	tgMessageClient   *tgclient.Client
	gatewayClient     *imgateway.Client
	mediaManager      sharedsvc.MediaManager
}

// HandleWebhook implements [provider.Provider].
func (p *Provider) HandleWebhook(ctx context.Context, payload []byte) error {
	gate, err := p.resolveGateFromContext(ctx)
	if err != nil {
		return err
	}

	if !gate.Enabled {
		return nil
	}

	update, err := p.unmarshalUpdate(payload)
	if err != nil {
		if errors.Is(err, noUpdateIDErr) {
			return nil
		}
		return err
	}

	ctx = withGatewayIdentity(ctx, gate)

	err = p.handleUpdate(ctx, gate, update)
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) unmarshalUpdate(payload []byte) (*model.WebhookUpdate, error) {
	var update model.WebhookUpdate
	if err := json.Unmarshal(payload, &update); err != nil {
		return nil, err
	}
	if update.UpdateID == 0 {
		return nil, noUpdateIDErr
	}

	return &update, nil
}

func (p *Provider) resolveGateFromContext(ctx context.Context) (*model.Gate, error) {
	uri := p.webhookURI(ctx)
	if uri == "" {
		return nil, errors.Internal("webhook uri not found")
	}

	gate, err := p.resolveByURI(ctx, uri)
	if err != nil {
		return nil, err
	}

	return gate, nil
}

func (p *Provider) resolveByURI(ctx context.Context, uri string) (*model.Gate, error) {
	gate, err := p.store.Get(ctx, &model.GetGateFilters{URI: &uri})
	if err != nil {
		return nil, err
	}
	return gate, nil
}

// ValidateSignature implements [provider.SignatureValidator].
// It validates the X-Telegram-Bot-Api-Secret-Token header Telegram sends on every
// webhook POST once a secret_token was set via setWebhook.
// https://core.telegram.org/bots/api#setwebhook
func (p *Provider) ValidateSignature(ctx context.Context, headers http.Header, body []byte) error {
	given := headers.Get(secretTokenHeader)
	if given == "" {
		return fmt.Errorf("missing %s header", secretTokenHeader)
	}

	gate, err := p.resolveGateFromContext(ctx)
	if err != nil {
		return err
	}
	if gate.WebhookSecret == "" {
		return fmt.Errorf("gate has no webhook secret configured")
	}
	if !hmac.Equal([]byte(given), []byte(gate.WebhookSecret)) {
		return fmt.Errorf("secret token mismatch")
	}
	return nil
}

func (p *Provider) webhookURI(ctx context.Context) string {
	uri, _ := ctx.Value(provider.WebhookURIKey).(string)
	if strings.HasPrefix(uri, "/") {
		uri, _ = strings.CutPrefix(uri, "/")
	}
	return uri
}

func (p *Provider) buildGateCacheKey(uri string) string {
	return fmt.Sprintf("%s/%s", p.Type(), uri)
}

func (p *Provider) handleUpdate(ctx context.Context, gate *model.Gate, update *model.WebhookUpdate) error {
	var (
		err      error
		updateID = update.UpdateID
	)
	if p.idempotencyCache.IsProcessed(gate.ID, updateID) {
		return nil
	}

	if update.Message != nil {
		// new incoming message
		err = p.handleNewMessage(ctx, gate, update.Message)
	} else if update.EditedMessage != nil {
		// existed message edited
	} else if update.CallbackQuery != nil {
		// callback query
		err = p.handleCallbackQuery(ctx, gate, update.CallbackQuery)
	} else if update.Poll != nil {
		// New poll state. Bots receive only updates about manually stopped polls and polls, which are sent by the bot.
	} else if update.PollAnswer != nil {
		// A user changed their answer in a non-anonymous poll.
		//  Bots receive new votes only in polls that were sent by the bot itself.
	}
	if err != nil {
		return err
	}

	p.idempotencyCache.MarkProcessed(gate.ID, updateID)

	return nil
}

func (p *Provider) fetchGate(ctx context.Context, gateID string) (*model.Gate, error) {
	parsed, err := uuid.Parse(gateID)
	if err != nil {
		return nil, err
	}

	return p.store.Get(ctx, &model.GetGateFilters{ID: &parsed})
}

// SendInteractive implements [provider.InteractiveSender].
func (p *Provider) SendInteractive(ctx context.Context, req *coremodel.Message) (*coremodel.MessageResponse, error) {
	gate, err := p.fetchGate(ctx, req.GateID)
	if err != nil {
		return nil, err
	}
	if req.Interactive == nil {
		return nil, errors.InvalidArgument("interactive is required")
	}

	var (
		keyboard    tgclient.OutgoingKeyboarder
		interactive = req.Interactive
		destination int64
	)

	destination, err = strconv.ParseInt(req.To.Sub, 10, 64)
	if err != nil {
		return nil, err
	}
	keyboard, err = getKeyboard(interactive)
	if err != nil {
		return nil, err
	}
	replyTo, err := p.getReplyTo(req)
	if err != nil {
		return nil, err
	}

	request := &tgclient.TextRequest{
		Text:            interactive.Body,
		ReplyMarkup:     keyboard,
		ChatID:          destination,
		ReplyParameters: replyTo,
	}

	msg, err := p.tgMessageClient.SendText(ctx, gate.Token, request)
	if err != nil {
		return nil, err
	}
	return &coremodel.MessageResponse{
		ID: strconv.FormatInt(msg.MessageID, 10),
	}, nil
}

func getKeyboard(interactive *coremodel.Interactive) (tgclient.OutgoingKeyboarder, error) {
	var (
		keyboard tgclient.OutgoingKeyboarder
		err      error
	)
	if interactive == nil {
		return nil, nil
	}
	if interactive.Markup != nil {
		keyboard, err = parseInlineKeyboard(interactive.Markup)
		if err != nil {
			return nil, err
		}
	} else if interactive.ListReply != nil {
		keyboard, err = parseReplyKeyboard(interactive.ListReply, interactive.SingleUse)
		if err != nil {
			return nil, err
		}
	}

	return keyboard, nil
}

func (p *Provider) getReplyTo(msg *coremodel.Message) (*model.ReplyParameters, error) {
	if msg.ReplyToExternalID != "" {
		msgID, err := strconv.ParseInt(msg.ReplyToExternalID, 10, 64)
		if err != nil {
			return nil, err
		}
		return &model.ReplyParameters{
			MessageID: &msgID,
		}, nil
	}
	return nil, nil
}

var _ provider.InteractiveSender = (*Provider)(nil)

// SendReaction implements [provider.ReactionSender].
func (p *Provider) SendReaction(ctx context.Context, req *provider.ReactionRequest) error {
	gate, err := p.fetchGate(ctx, req.GateID)
	if err != nil {
		return err
	}

	if gate == nil {
		return errors.NotFound("gate not found")
	}

	chatID, err := strconv.ParseInt(req.ExternalID, 10, 64)
	if err != nil {
		return err
	}

	messageID, err := strconv.ParseInt(req.ExternalMessageID, 10, 64)
	if err != nil {
		return err
	}

	var reaction []tgclient.ReactionEmoji
	if !req.Removed && req.Emoji != "" {
		reaction = []tgclient.ReactionEmoji{
			{
				Type:  "emoji",
				Emoji: req.Emoji,
			},
		}
	}

	return p.tgMessageClient.SetMessageReaction(ctx, gate.Token, &tgclient.ReactionRequest{
		ChatID:    chatID,
		MessageID: messageID,
		Reaction:  reaction,
	})
}

var _ provider.ReactionSender = (*Provider)(nil)

// SendDocument implements [provider.Provider].
func (p *Provider) SendDocument(ctx context.Context, req *coremodel.Message) (*coremodel.MessageResponse, error) {
	gate, err := p.fetchGate(ctx, req.GateID)
	if err != nil {
		return nil, err
	}
	if gate == nil {
		return nil, errors.NotFound("gate not found")
	}
	if len(req.Documents) == 0 {
		return nil, errors.InvalidArgument("no documents")
	}

	destination, err := strconv.ParseInt(req.To.Sub, 10, 64)
	if err != nil {
		return nil, err
	}
	keyboard, err := getKeyboard(req.Interactive)
	if err != nil {
		return nil, err
	}
	replyTo, err := p.getReplyTo(req)
	if err != nil {
		return nil, err
	}
	var (
		msgs           = make(map[string]any, len(req.Documents))
		firstMessageID string
	)
	for _, doc := range req.Documents {
		request := &tgclient.DocumentRequest{
			ReplyMarkup:     keyboard,
			ChatID:          destination,
			Document:        doc.URL,
			Caption:         &req.Text,
			ReplyParameters: replyTo,
		}
		msg, err := p.tgMessageClient.SendDocument(ctx, gate.Token, request)
		if err != nil {
			return nil, err
		}
		if firstMessageID == "" {
			firstMessageID = strconv.FormatInt(msg.MessageID, 10)
		}
		msgs[doc.ID] = strconv.FormatInt(msg.MessageID, 10)
	}
	return &coremodel.MessageResponse{ID: firstMessageID, MD: msgs}, nil
}

// SendImage implements [provider.Provider].
func (p *Provider) SendImage(ctx context.Context, req *coremodel.Message) (*coremodel.MessageResponse, error) {
	gate, err := p.fetchGate(ctx, req.GateID)
	if err != nil {
		return nil, err
	}
	if gate == nil {
		return nil, errors.NotFound("gate not found")
	}
	if len(req.Images) == 0 {
		return nil, errors.InvalidArgument("no images")
	}

	destination, err := strconv.ParseInt(req.To.Sub, 10, 64)
	if err != nil {
		return nil, err
	}
	keyboard, err := getKeyboard(req.Interactive)
	if err != nil {
		return nil, err
	}
	replyTo, err := p.getReplyTo(req)
	if err != nil {
		return nil, err
	}

	var (
		msgs           = make(map[string]any, len(req.Images))
		firstMessageID string
	)
	for _, img := range req.Images {
		request := &tgclient.PhotoRequest{
			ReplyMarkup:     keyboard,
			ChatID:          destination,
			Photo:           img.URL,
			Caption:         &req.Text,
			ReplyParameters: replyTo,
		}
		msg, err := p.tgMessageClient.SendPhoto(ctx, gate.Token, request)
		if err != nil {
			return nil, err
		}
		if firstMessageID == "" {
			firstMessageID = strconv.FormatInt(msg.MessageID, 10)
		}
		msgs[img.ID] = strconv.FormatInt(msg.MessageID, 10)
	}
	return &coremodel.MessageResponse{ID: firstMessageID, MD: msgs}, nil
}

// SendText implements [provider.Provider].
func (p *Provider) SendText(ctx context.Context, req *coremodel.Message) (*coremodel.MessageResponse, error) {
	gate, err := p.fetchGate(ctx, req.GateID)
	if err != nil {
		return nil, err
	}
	if gate == nil {
		return nil, errors.NotFound("gate not found")
	}

	destination, err := strconv.ParseInt(req.To.Sub, 10, 64)
	if err != nil {
		return nil, err
	}
	keyboard, err := getKeyboard(req.Interactive)
	if err != nil {
		return nil, err
	}
	replyTo, err := p.getReplyTo(req)
	if err != nil {
		return nil, err
	}

	request := &tgclient.TextRequest{
		Text:            req.Text,
		ReplyMarkup:     keyboard,
		ChatID:          destination,
		ReplyParameters: replyTo,
	}

	msg, err := p.tgMessageClient.SendText(ctx, gate.Token, request)
	if err != nil {
		return nil, err
	}
	return &coremodel.MessageResponse{
		ID: strconv.FormatInt(msg.MessageID, 10),
	}, nil
}

// Type implements [provider.Provider].
func (p *Provider) Type() string {
	return providerType
}
