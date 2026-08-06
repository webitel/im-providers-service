package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"

	"github.com/webitel/im-providers-service/config"
	vibmodel "github.com/webitel/im-providers-service/internal/viber/model"
	vibstore "github.com/webitel/im-providers-service/internal/viber/store"
)

var _ ViberManager = (*ViberService)(nil)

type ViberManager interface {
	CreateGate(ctx context.Context, req vibmodel.CreateViber) (*vibmodel.ViberGate, error)
	GetGate(ctx context.Context, id string) (*vibmodel.ViberGate, error)
	UpdateGate(ctx context.Context, req vibmodel.UpdateViber) (*vibmodel.ViberGate, error)
	DeleteGate(ctx context.Context, id string) (*vibmodel.ViberGate, error)
}

type ProviderAPI interface {
	GetAccountInfo(ctx context.Context, token string) (*vibmodel.AccountInfo, error)
	SetWebhook(ctx context.Context, token, url string) error
	RemoveWebhook(ctx context.Context, token string) error
}

type ViberService struct {
	store vibstore.ViberStore
	api   ProviderAPI
	cfg   *config.Config
	log   *slog.Logger
}

func NewViberService(store vibstore.ViberStore, api ProviderAPI, cfg *config.Config, log *slog.Logger) *ViberService {
	return &ViberService{
		store: store,
		api:   api,
		cfg:   cfg,
		log:   log.With("layer", "service", "domain", "viber_gate"),
	}
}

func (s *ViberService) CreateGate(ctx context.Context, req vibmodel.CreateViber) (*vibmodel.ViberGate, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	info, err := s.api.GetAccountInfo(ctx, req.AuthToken)
	if err != nil {
		return nil, err
	}

	senderName := req.SenderName
	if senderName == "" {
		senderName = info.Name
	}
	senderAvatar := req.SenderAvatar
	if senderAvatar == "" {
		senderAvatar = info.Avatar
	}

	webhookURI, err := genWebhookURI()
	if err != nil {
		return nil, fmt.Errorf("viber: generate webhook uri: %w", err)
	}

	gate := &vibmodel.ViberGate{
		Name:         req.Name,
		Peer:         req.Peer,
		BotID:        info.ID,
		BotURI:       info.URI,
		AuthToken:    req.AuthToken,
		SenderName:   senderName,
		SenderAvatar: senderAvatar,
		WebhookURI:   webhookURI,
		Enabled:      true,
	}

	if err := s.store.Insert(ctx, req.Dc, gate); err != nil {
		s.log.Error("failed to create viber gate", "bot_id", info.ID, "err", err)
		return nil, err
	}

	if err := s.api.SetWebhook(ctx, req.AuthToken, s.webhookURL(webhookURI)); err != nil {
		s.log.Error("set_webhook failed, rolling back gate", "id", gate.ID, "err", err)
		if unbindErr := s.store.Unbind(ctx, gate.ID); unbindErr != nil {
			s.log.Error("rollback failed", "id", gate.ID, "err", unbindErr)
		}
		return nil, err
	}

	s.log.Info("viber gate created", "id", gate.ID, "bot_id", info.ID)
	return gate, nil
}

func (s *ViberService) GetGate(ctx context.Context, id string) (*vibmodel.ViberGate, error) {
	return s.store.Select(ctx, id)
}

func (s *ViberService) UpdateGate(ctx context.Context, req vibmodel.UpdateViber) (*vibmodel.ViberGate, error) {
	gate, err := s.store.Select(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	req.ApplyTo(gate)

	if err := s.store.Update(ctx, gate); err != nil {
		s.log.Error("failed to update viber gate", "id", req.ID, "err", err)
		return nil, err
	}

	s.log.Info("viber gate updated", "id", gate.ID)
	return gate, nil
}

func (s *ViberService) DeleteGate(ctx context.Context, id string) (*vibmodel.ViberGate, error) {
	gate, err := s.store.Select(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.api.RemoveWebhook(ctx, gate.AuthToken); err != nil {
		s.log.Warn("remove_webhook failed", "id", id, "err", err)
	}

	if err := s.store.Unbind(ctx, id); err != nil {
		s.log.Error("failed to unbind viber gate", "id", id, "err", err)
		return nil, err
	}

	s.log.Warn("viber gate configuration removed", "id", id, "bot_id", gate.BotID)
	return gate, nil
}

func (s *ViberService) webhookURL(uri string) string {
	base := strings.TrimRight(s.cfg.Service.PublicURL, "/")
	path := "/" + strings.Trim(s.cfg.Service.WebhookPath, "/")
	return fmt.Sprintf("%s%s/viber/%s", base, path, uri)
}

func genWebhookURI() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
