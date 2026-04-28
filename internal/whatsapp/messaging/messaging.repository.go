package messaging

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/webitel/im-providers-service/infra/db/postgresx"
	"github.com/webitel/webitel-go-kit/pkg/errors"
)

type messagingRepository struct {
	db postgresx.DB
}

func newMessagingRepository(db postgresx.DB) *messagingRepository {
	return &messagingRepository{db: db}
}

func (repository *messagingRepository) Resolve(ctx context.Context, query ResolveWhatsAppBusinessAccountQuery) (*WhatsAppBusinessAccount, error) {
	stmt, args := repository.prepareResolveWhatsAppBusinessAccountQuery(query)

	rows, err := repository.db.Replica().Query(ctx, stmt, args)
	if err != nil {
		return nil, errors.Internal(
			"executing resolve whatsapp business account query",
			errors.WithCause(err),
			errors.WithID("messaging.repository.resolve"),
			errors.WithValue("stmt", postgresx.CompactSQL(stmt)),
			errors.WithValue("args", query),
		)
	}

	whatsAppBusinessAccount, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByNameLax[WhatsAppBusinessAccount])
	if err != nil {
		return nil, errors.Internal("collecting whatsapp business acoount record", errors.WithCause(err), errors.WithID("messaging.repository.resolve"))
	}

	return whatsAppBusinessAccount, nil
}

func (repository *messagingRepository) prepareResolveWhatsAppBusinessAccountQuery(query ResolveWhatsAppBusinessAccountQuery) (string, postgresx.NamedArgs) {

	return "", nil
}
