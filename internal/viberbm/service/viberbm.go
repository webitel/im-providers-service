package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	vibbmmodel "github.com/webitel/im-providers-service/internal/viberbm/model"
	vibbmstore "github.com/webitel/im-providers-service/internal/viberbm/store"
)

var _ ViberBMManager = (*ViberBMService)(nil)

// ViberBMManager is the business surface for Viber BM gateway CRUD plus the
// cold-start template-send primitive.
type ViberBMManager interface {
	CreateGate(ctx context.Context, req vibbmmodel.CreateViberBM) (*vibbmmodel.ViberBMGate, error)
	GetGate(ctx context.Context, id string) (*vibbmmodel.ViberBMGate, error)
	UpdateGate(ctx context.Context, req vibbmmodel.UpdateViberBM) (*vibbmmodel.ViberBMGate, error)
	DeleteGate(ctx context.Context, id string) (*vibbmmodel.ViberBMGate, error)
	ListGates(ctx context.Context, dc int64, q string, page, size int) ([]*vibbmmodel.ViberBMGate, bool, error)
	SendTemplate(ctx context.Context, gateID, to, templateID, language string, params map[string]string) (*sharedmodel.MessageResponse, error)
}

// TemplateSender is the cold-start send surface, satisfied by the provider
// adapter. Kept as a narrow consumer-side interface so the service does not
// depend on the whole provider package.
type TemplateSender interface {
	SendTemplate(ctx context.Context, gateID, to, templateID, language string, params map[string]string) (*sharedmodel.MessageResponse, error)
}

type ViberBMService struct {
	store  vibbmstore.ViberBMStore
	sender TemplateSender
	log    *slog.Logger
}

func NewViberBMService(store vibbmstore.ViberBMStore, sender TemplateSender, log *slog.Logger) *ViberBMService {
	return &ViberBMService{
		store:  store,
		sender: sender,
		log:    log.With("layer", "service", "domain", "viber_bm_gate"),
	}
}

func (s *ViberBMService) CreateGate(ctx context.Context, req vibbmmodel.CreateViberBM) (*vibbmmodel.ViberBMGate, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	webhookURI, err := genWebhookURI()
	if err != nil {
		return nil, fmt.Errorf("viber_bm: generate webhook uri: %w", err)
	}

	gate := &vibbmmodel.ViberBMGate{
		Name:          req.Name,
		Peer:          req.Peer,
		BaseURL:       req.BaseURL,
		SenderName:    req.SenderName,
		APIKey:        req.APIKey,
		WebhookSecret: req.WebhookSecret,
		WebhookURI:    webhookURI,
		Enabled:       true,
	}

	if err := s.store.Insert(ctx, req.Dc, gate); err != nil {
		s.log.Error("failed to create viber_bm gate", "sender_name", req.SenderName, "err", err)

		return nil, err
	}

	s.log.Info("viber_bm gate created", "id", gate.ID, "sender_name", gate.SenderName)

	return gate, nil
}

func (s *ViberBMService) GetGate(ctx context.Context, id string) (*vibbmmodel.ViberBMGate, error) {
	return s.store.Select(ctx, id)
}

func (s *ViberBMService) UpdateGate(ctx context.Context, req vibbmmodel.UpdateViberBM) (*vibbmmodel.ViberBMGate, error) {
	gate, err := s.store.Select(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	req.ApplyTo(gate)

	if err := s.store.Update(ctx, gate); err != nil {
		s.log.Error("failed to update viber_bm gate", "id", req.ID, "err", err)

		return nil, err
	}

	s.log.Info("viber_bm gate updated", "id", gate.ID)

	return gate, nil
}

func (s *ViberBMService) DeleteGate(ctx context.Context, id string) (*vibbmmodel.ViberBMGate, error) {
	gate, err := s.store.Select(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.store.Unbind(ctx, id); err != nil {
		s.log.Error("failed to unbind viber_bm gate", "id", id, "err", err)

		return nil, err
	}

	s.log.Warn("viber_bm gate configuration removed", "id", id, "sender_name", gate.SenderName)

	return gate, nil
}

func (s *ViberBMService) ListGates(ctx context.Context, dc int64, q string, page, size int) ([]*vibbmmodel.ViberBMGate, bool, error) {
	return s.store.List(ctx, dc, q, page, size)
}

func (s *ViberBMService) SendTemplate(ctx context.Context, gateID, to, templateID, language string, params map[string]string) (*sharedmodel.MessageResponse, error) {
	return s.sender.SendTemplate(ctx, gateID, to, templateID, language, params)
}

func genWebhookURI() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return hex.EncodeToString(buf), nil
}
