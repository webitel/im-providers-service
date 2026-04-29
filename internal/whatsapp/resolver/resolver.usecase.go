package resolver

import (
	"context"
	"log/slog"

	"github.com/webitel/im-providers-service/internal/whatsapp/common"
	"github.com/webitel/webitel-go-kit/pkg/errors"
)

var ErrNoCorespondingWhatsAppBusinessAccount = errors.NotFound("zero enabled gates found for coresponding whatsapp business account phone id")

type ResolveWhatsAppBusinessAccountQuery interface {
	GetPhoneNumberID() string
}

type ResolverRepository interface {
	Resolve(ctx context.Context, query resolveWhatsAppBusinessAccountQuery) (*common.WhatsappBusinessAccount, error)
}

type resolver[T ResolveWhatsAppBusinessAccountQuery] struct {
	logger     *slog.Logger
	repository ResolverRepository
}

func newResolver[T ResolveWhatsAppBusinessAccountQuery](logger *slog.Logger, repository ResolverRepository) *resolver[T] {
	log := logger.With("component", "whatsapp.resolver")
	return &resolver[T]{
		logger:     log,
		repository: repository,
	}
}

func (resolver *resolver[T]) Resolve(ctx context.Context, query T) (*common.WhatsappBusinessAccount, error) {
	log := resolver.logger.With("operation", "resolve")
	resolveQuery := resolveWhatsAppBusinessAccountQuery{PhoneNumberID: query.GetPhoneNumberID()}

	whatsAppBusinessAccount, err := resolver.repository.Resolve(ctx, resolveQuery)
	if err != nil {
		if errors.Is(err, ErrNoCorespondingWhatsAppBusinessAccount) {
			return nil, nil
		}

		log.Error("resolving whatsapp business account", "error", err)
		return nil, errors.Wrap(err, errors.WithID("whatsapp.resolver.usecase.resolve"))
	}

	return whatsAppBusinessAccount, nil
}
