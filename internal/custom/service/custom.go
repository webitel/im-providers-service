package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"

	"github.com/webitel/im-providers-service/config"
	custommodel "github.com/webitel/im-providers-service/internal/custom/model"
	customstore "github.com/webitel/im-providers-service/internal/custom/store"
)

var _ CustomManager = (*CustomService)(nil)

type CustomManager interface {
	CreateGate(ctx context.Context, req custommodel.CreateCustom) (*custommodel.CustomGate, error)
	GetGate(ctx context.Context, id string) (*custommodel.CustomGate, error)
	UpdateGate(ctx context.Context, req custommodel.UpdateCustom) (*custommodel.CustomGate, error)
	DeleteGate(ctx context.Context, id string) (*custommodel.CustomGate, error)
	WebhookURL(uri string) string
}

type CustomService struct {
	store customstore.CustomStore
	cfg   *config.Config
	log   *slog.Logger
}

func NewCustomService(store customstore.CustomStore, cfg *config.Config, log *slog.Logger) *CustomService {
	return &CustomService{
		store: store,
		cfg:   cfg,
		log:   log.With("layer", "service", "domain", "custom_gate"),
	}
}

func (s *CustomService) CreateGate(ctx context.Context, req custommodel.CreateCustom) (*custommodel.CustomGate, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	webhookURI, err := genWebhookURI()
	if err != nil {
		return nil, fmt.Errorf("custom: generate webhook uri: %w", err)
	}

	secret := req.AppSecret
	if secret == "" {
		if secret, err = genAppSecret(); err != nil {
			return nil, fmt.Errorf("custom: generate app secret: %w", err)
		}
	}

	timeout := req.RequestTimeoutMS
	if timeout <= 0 {
		timeout = custommodel.DefaultRequestTimeoutMS
	}

	retries := req.RetryAttempts
	if retries <= 0 {
		retries = custommodel.DefaultRetryAttempts
	}

	gate := &custommodel.CustomGate{
		Name:             req.Name,
		Peer:             req.Peer,
		CallbackURL:      req.CallbackURL,
		AppSecret:        secret,
		WebhookURI:       webhookURI,
		AllowedIPs:       req.AllowedIPs,
		RequestTimeoutMS: timeout,
		RetryAttempts:    retries,
		Enabled:          true,
	}

	if err := s.store.Insert(ctx, req.Dc, gate); err != nil {
		s.log.Error("failed to create custom gate", "name", req.Name, "err", err)

		return nil, err
	}

	s.log.Info("custom gate created", "id", gate.ID)

	return gate, nil
}

func (s *CustomService) GetGate(ctx context.Context, id string) (*custommodel.CustomGate, error) {
	return s.store.Select(ctx, id)
}

func (s *CustomService) UpdateGate(ctx context.Context, req custommodel.UpdateCustom) (*custommodel.CustomGate, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	gate, err := s.store.Select(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	req.ApplyTo(gate)

	if err := s.store.Update(ctx, gate); err != nil {
		s.log.Error("failed to update custom gate", "id", req.ID, "err", err)

		return nil, err
	}

	s.log.Info("custom gate updated", "id", gate.ID)

	return gate, nil
}

func (s *CustomService) DeleteGate(ctx context.Context, id string) (*custommodel.CustomGate, error) {
	gate, err := s.store.Select(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.store.Unbind(ctx, id); err != nil {
		s.log.Error("failed to unbind custom gate", "id", id, "err", err)

		return nil, err
	}

	s.log.Warn("custom gate configuration removed", "id", id)

	return gate, nil
}

func (s *CustomService) WebhookURL(uri string) string {
	if uri == "" {
		return ""
	}

	base := strings.TrimRight(s.cfg.Service.PublicURL, "/")
	path := "/" + strings.Trim(s.cfg.Service.WebhookPath, "/")

	return fmt.Sprintf("%s%s/%s/%s", base, path, custommodel.ProviderType, uri)
}

func genWebhookURI() (string, error) {
	return randomHex(24)
}

func genAppSecret() (string, error) {
	return randomHex(32)
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return hex.EncodeToString(buf), nil
}
