package resolver

import (
	"log/slog"

	"github.com/webitel/im-providers-service/infra/db/postgresx"
)

type resolverContainer[T ResolveWhatsAppBusinessAccountQuery] struct {
	Resolver *resolver[T]
}

func NewResolverModule[T ResolveWhatsAppBusinessAccountQuery](logger *slog.Logger, db postgresx.DB) *resolverContainer[T] {
	var (
		resolverRepository = newResolverRepository(db)
		resolverUseCase    = newResolver[T](logger, resolverRepository)
	)

	return &resolverContainer[T]{
		Resolver: resolverUseCase,
	}
}
