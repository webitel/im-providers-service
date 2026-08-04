package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/webitel/im-providers-service/infra/auth"
	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
	"github.com/webitel/im-providers-service/internal/telegram/bot/client"
	"github.com/webitel/im-providers-service/internal/telegram/bot/model"
	"github.com/webitel/im-providers-service/internal/telegram/bot/store"
)

// Options carries the config values TelegramBotService needs to build a
// webhook URL to register with Telegram's setWebhook. Kept narrow (rather
// than injecting *config.Config) to match sibling services.
type Options struct {
	PublicURL   string
	WebhookPath string
}

func NewTelegramBotService(store store.TelegramBotStore, cache sharedstore.GateCache, tg *client.Client, opts Options, log *slog.Logger) *TelegramBotService {
	return &TelegramBotService{
		store: store,
		cache: cache,
		tg:    tg,
		opts:  opts,
		log:   log,
	}
}

type TelegramBotService struct {
	store store.TelegramBotStore
	cache sharedstore.GateCache

	tg   *client.Client
	opts Options
	log  *slog.Logger
}

// webhookURL builds the fully-qualified HTTPS URL Telegram should deliver
// updates to, matching the route pattern registered in cmd/fx.go's ProvideRouter
// (<WebhookPath>/{provider}/{uri}).
func (s *TelegramBotService) webhookURL(uri string) (string, error) {
	base := strings.TrimRight(s.opts.PublicURL, "/")
	if base == "" {
		return "", errors.New("service.public_url is required to register telegram webhooks")
	}
	path := "/" + strings.Trim(s.opts.WebhookPath, "/")
	return fmt.Sprintf("%s%s/%s/%s", base, path, model.ProviderType, uri), nil
}

func (s *TelegramBotService) CreateTelegramBot(ctx context.Context, req *model.CreateGate) (*model.Gate, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	identity, ok := auth.GetIdentityFromContext(ctx)
	if !ok {
		return nil, errors.New("identity not found")
	}
	req.DC = identity.GetDomainID()
	if req.Name == "" {
		return nil, errors.New("name is required")
	}
	if req.Token == "" {
		return nil, errors.New("token is required")
	}
	if req.Bot == nil {
		return nil, errors.New("bot is required")
	}
	if req.URI == "" {
		return nil, errors.New("uri is required")
	}
	if req.WebhookSecret == "" {
		return nil, errors.New("webhook_secret is required")
	}

	result, err := s.store.Insert(ctx, &model.CreateGate{ // copy to not modify arg
		Name:          req.Name,
		DC:            identity.GetDomainID(),
		Bot:           req.Bot,
		Enabled:       req.Enabled,
		Token:         req.Token,
		URI:           req.URI,
		WebhookSecret: req.WebhookSecret,
	})
	if err != nil {
		return nil, err
	}

	url, err := s.webhookURL(result.URI)
	if err != nil {
		s.rollbackGate(ctx, result)
		return nil, err
	}
	if err := s.tg.SetWebhook(ctx, result.Token, url, result.WebhookSecret); err != nil {
		s.rollbackGate(ctx, result)
		return nil, err
	}

	return result, nil
}

// rollbackGate best-effort deletes a just-created gate after webhook
// registration fails, so a gate only exists in the DB if Telegram actually
// accepted the webhook.
func (s *TelegramBotService) rollbackGate(ctx context.Context, gate *model.Gate) {
	if err := s.store.Delete(ctx, gate.ID, gate.DC); err != nil {
		s.log.Error("failed to roll back telegram gate after webhook registration failure",
			"gate_id", gate.ID, "error", err)
	}
}

func (s *TelegramBotService) UpdateTelegramBot(ctx context.Context, req *model.UpdateGate) (*model.Gate, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	if req.ID == uuid.Nil {
		return nil, errors.New("id is required")
	}
	identity, ok := auth.GetIdentityFromContext(ctx)
	if !ok {
		return nil, errors.New("identity not found")
	}
	res, err := s.store.Update(ctx, &model.UpdateGate{
		ID:            req.ID,
		DC:            identity.GetDomainID(),
		Name:          req.Name,
		Token:         req.Token,
		Bot:           req.Bot,
		Enabled:       req.Enabled,
		WebhookSecret: req.WebhookSecret,
	})
	if err != nil {
		return nil, err
	}

	switch {
	case req.Enabled != nil && !*req.Enabled:
		// Explicit disable -> deregister. Best-effort: the gate is already inert at the
		// app layer (HandleWebhook ignores disabled gates), so a lingering Telegram-side
		// registration is low severity.
		if err := s.tg.DeleteWebhook(ctx, res.Token); err != nil {
			s.log.Warn("failed to unregister telegram webhook after disabling bot",
				"gate_id", res.ID, "error", err)
		}
	case res.Enabled && (req.Token != nil || (req.Enabled != nil && *req.Enabled)):
		// Becoming enabled, or token rotated while enabled -> (re)register.
		url, err := s.webhookURL(res.URI)
		if err != nil {
			return nil, err
		}
		if err := s.tg.SetWebhook(ctx, res.Token, url, res.WebhookSecret); err != nil {
			return nil, err
		}
	}

	return res, nil
}

func (s *TelegramBotService) DeleteTelegramBot(ctx context.Context, id uuid.UUID) (*model.Gate, error) {
	if id == uuid.Nil {
		return nil, errors.New("id is required")
	}
	identity, ok := auth.GetIdentityFromContext(ctx)
	if !ok {
		return nil, errors.New("identity not found")
	}
	if identity.GetDomainID() == 0 {
		return nil, errors.New("domain id is required")
	}
	domainID := identity.GetDomainID()

	gate, err := s.store.Get(ctx, &model.GetGateFilters{ID: &id, DC: &domainID})
	if err != nil {
		s.log.Warn("failed to fetch telegram gate before webhook unregistration", "gate_id", id, "error", err)
	} else if err := s.tg.DeleteWebhook(ctx, gate.Token); err != nil {
		// A Telegram-side hiccup shouldn't prevent the user from removing their own gate row.
		s.log.Warn("failed to unregister telegram webhook", "gate_id", id, "error", err)
	}

	if err := s.store.Delete(ctx, id, domainID); err != nil {
		return nil, err
	}

	return &model.Gate{}, nil
}

func (s *TelegramBotService) GetTelegramBot(ctx context.Context, id uuid.UUID) (*model.Gate, error) {
	if id == uuid.Nil {
		return nil, errors.New("id is required")
	}
	identity, ok := auth.GetIdentityFromContext(ctx)
	if !ok {
		return nil, errors.New("identity not found")
	}
	if identity.GetDomainID() == 0 {
		return nil, errors.New("domain id is required")
	}
	domainID := identity.GetDomainID()
	res, err := s.store.Get(ctx, &model.GetGateFilters{
		ID: &id,
		DC: &domainID,
	})
	if err != nil {
		return nil, err
	}

	return res, nil
}
