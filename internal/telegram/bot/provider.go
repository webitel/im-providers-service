package bot

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/redis/go-redis/v9"
	coremodel "github.com/webitel/im-providers-service/internal/core/model"
	sharedsvc "github.com/webitel/im-providers-service/internal/core/service"
	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
	"github.com/webitel/im-providers-service/internal/provider"
	"github.com/webitel/im-providers-service/internal/telegram/bot/model"
)

var _ provider.Provider = (*Provider)(nil)

const (
	providerType = "telegram_https"
)

func New(
	l *slog.Logger,
	gc sharedstore.GateCache,
	uc sharedstore.ExternalUserCache,
	media sharedsvc.MediaManager,
	rdb *redis.Client,

) *Provider {
	return &Provider{}
}

type Provider struct {
}

// HandleWebhook implements [provider.Provider].
func (p *Provider) HandleWebhook(ctx context.Context, payload []byte) error {
	uri := p.webhookURI(ctx)

	var update model.WebhookUpdate
	if err := json.Unmarshal(payload, &update); err != nil {
		return err
	}

}

func (p *Provider) webhookURI(ctx context.Context) string {
	uri, _ := ctx.Value(provider.WebhookURIKey).(string)
	if !strings.HasPrefix(uri, "/") {
		return "/" + uri
	}
	return uri
}

// SendInteractive implements [provider.InteractiveSender].
func (p *Provider) SendInteractive(ctx context.Context, req *coremodel.Message) (*coremodel.MessageResponse, error) {
	panic("unimplemented")
}

var _ provider.InteractiveSender = (*Provider)(nil)

// SendDocument implements [provider.Provider].
func (p *Provider) SendDocument(ctx context.Context, req *coremodel.Message) (*coremodel.MessageResponse, error) {
	panic("unimplemented")
}

// SendImage implements [provider.Provider].
func (p *Provider) SendImage(ctx context.Context, req *coremodel.Message) (*coremodel.MessageResponse, error) {
	panic("unimplemented")
}

// SendText implements [provider.Provider].
func (p *Provider) SendText(ctx context.Context, req *coremodel.Message) (*coremodel.MessageResponse, error) {
	panic("unimplemented")
}

// Type implements [provider.Provider].
func (p *Provider) Type() string {
	return providerType
}
