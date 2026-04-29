package resolver

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/webitel/im-providers-service/infra/db/postgresx"
	"github.com/webitel/im-providers-service/internal/whatsapp/common"
	"github.com/webitel/webitel-go-kit/pkg/errors"
)

type resolverRepository struct {
	db postgresx.DB
}

func newResolverRepository(db postgresx.DB) *resolverRepository {
	return &resolverRepository{db: db}
}

func (repository *resolverRepository) Resolve(ctx context.Context, query resolveWhatsAppBusinessAccountQuery) (*common.WhatsappBusinessAccount, error) {
	args := postgresx.NamedArgs{
		"PhoneNumberID": query.PhoneNumberID,
	}

	rows, err := repository.db.Replica().Query(ctx, resolveWhatsAppBusinessAccountSQLQuery, args)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoCorespondingWhatsAppBusinessAccount
		}
		return nil, errors.Internal(
			"executing resolve whatsapp business account sql query",
			errors.WithCause(err),
			errors.WithID("whatsapp.resolver.repository.resolve"),
			errors.WithValue("stmt", resolveWhatsAppBusinessAccountSQLQuery),
			errors.WithValue("phone_number_id", query.PhoneNumberID),
		)
	}

	whatsAppBusinessAccount, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByNameLax[common.WhatsappBusinessAccount])
	if err != nil {
		return nil, errors.Internal(
			"collecting resolved whatsapp business account",
			errors.WithCause(err),
			errors.WithID("whatsapp.resolver.repository.resolve"),
			errors.WithValue("phone_number_id", query.PhoneNumberID),
		)
	}

	return whatsAppBusinessAccount, nil
}

var resolveWhatsAppBusinessAccountSQLQuery = postgresx.CompactSQL(`
	select
		"gw"."id" as "id",
		"g"."dc" as "dc",
		"gw"."phone_number" as "phone_number",
		"gw"."phone_number_id" as "phone_number_id",
		"gw"."access_token" as "access_token_encrypted",
		"gw"."access_token_expires_at" as "access_token_expires_at",
		"gw"."business_id" as "business_id",
		to_jsonb("b".*) as "bot"
	from "im_provider"."gate_waba" "gw"
	inner join "im_provider"."gates" "g" using("id")
	inner join "im_provider"."bots" "b" using("id")
	where "gw" = @PhoneNumberID and "g"."enabled"
	limit 1;
`)
