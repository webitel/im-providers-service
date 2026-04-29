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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.NotFound(
				"no whatsapp gates that are enabled and have coresponding binded contact",
				errors.WithCause(err),
				errors.WithID("messaging.repository.resolve"),
				errors.WithValue("bot_iss", query.BotIss),
				errors.WithValue("bot_sub", query.BotSub),
			)
		}

		return nil, errors.Internal("collecting whatsapp business acoount record", errors.WithCause(err), errors.WithID("messaging.repository.resolve"))
	}

	return whatsAppBusinessAccount, nil
}

func (repository *messagingRepository) prepareResolveWhatsAppBusinessAccountQuery(query ResolveWhatsAppBusinessAccountQuery) (string, postgresx.NamedArgs) {
	stmt := `
		select
			"gw"."phone_number" as "phone_number",
			"gw"."phone_number_id" as "phone_number_id",
			"gw"."business_id" as "business_id",
			"gw"."access_token" as "access_token",
			"gw"."access_token_expires_at" as "access_token_expires_at",
			to_jsonb("bc".*) as "contact"
		from "im_provider"."binded_contact" "bc"
		inner join "im_provider"."gates" "g" using("id")
		inner join "im_provider"."gate_waba" "gw" using("id")
		where ("bc"."iss", "bc"."sub")=(@Iss,@Sub)
			and "g"."enabled"
		limit 1;
	`

	args := postgresx.NamedArgs{
		"Iss": query.BotIss,
		"Sub": query.BotSub,
	}

	return stmt, args
}
