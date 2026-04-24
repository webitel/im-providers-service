package gate

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/webitel/im-providers-service/infra/db/postgresx"
	"github.com/webitel/webitel-go-kit/pkg/errors"
	"google.golang.org/grpc/codes"
)

type gateRepository struct {
	db postgresx.DB
}

func newGateRepository(db postgresx.DB) *gateRepository {
	return &gateRepository{
		db: db,
	}
}

func (repository *gateRepository) Save(ctx context.Context, wabaGate *Gate) (*Gate, error) {
	stmt, args := repository.prepareWhatsAppGateSaveQuery(wabaGate)

	rows, err := repository.db.Query(ctx, stmt, args)
	if err != nil {
		if pgErr := err.(*pgconn.PgError); pgErr != nil {
			switch pgErr.Code {
			case postgresx.CodeCheckViolation:
				return nil, errors.InvalidArgument("check violation for new gate record", errors.WithID("gate.repository.save"), errors.WithCause(err))
			case postgresx.CodeUniqueViolation:
				return nil, errors.New("conflict: whatsapp gate for this settings already exists", errors.WithID("gate.repository.save"), errors.WithCause(err), errors.WithCode(codes.AlreadyExists))
			case postgresx.CodeForeignKeyViolation:
				return nil, errors.InvalidArgument("meta app for given id not exists", errors.WithCause(err), errors.WithID("gate.repository.save"), errors.WithValue("meta_app_id", wabaGate.WhatsAppBusinessAccountGate.MetaAppID.String()))
			}
		}

		return nil, errors.Internal("performing database save", errors.WithCause(err), errors.WithID("gate.repository.save"), errors.WithValue("stmt", stmt))
	}

	record, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByNameLax[Gate])
	if err != nil {
		return nil, errors.Internal("collecting saved record", errors.WithCause(err), errors.WithID("gate.repository.save"))
	}

	return record, nil
}

func (repository *gateRepository) prepareWhatsAppGateSaveQuery(wabaGate *Gate) (string, postgresx.NamedArgs) {
	stmt := `
		with gate_insert as (
			insert into "im_provider"."gates" (
				"name", "type", "enabled"
			)
			values (
				@Name, @Type, @Enabled
			)
			returning
				"id", "name", "type", "enabled", "created_at", "updated_at"
		),
		waba_gate_ins as (
			insert into "im_provider"."gate_waba" (
				"id", "meta_app_id", "phone_number", "phone_number_id", "access_token",
				"access_token_expires_at", "business_id"
			)
			select
				"id",
				@MetaAppID,
				@PhoneNumber,
				@PhoneNumberID,
				@AccessToken,
				@AccessTokenExpiresAt,
				@BusinessID
			from gate_insert
		)
		select
			g.id as id,
			g.name as name,
			g.type as type,
			g.enabled as enabled,
			g.created_at as created_at,
			g.updated_at as updated_at,
			to_jsonb(w.*) as whats_app_business_account_gate
		from gate_insert g
		inner join waba_gate_ins w using(id)
	`

	args := postgresx.NamedArgs{
		"Name":                 wabaGate.Name,
		"Type":                 wabaGate.Type,
		"Enabled":              wabaGate.Enabled,
		"MetaAppID":            wabaGate.WhatsAppBusinessAccountGate.MetaAppID,
		"PhoneNumber":          wabaGate.WhatsAppBusinessAccountGate.PhoneNumber,
		"PhoneNumberID":        wabaGate.WhatsAppBusinessAccountGate.PhoneNumberID,
		"AccessToken":          wabaGate.WhatsAppBusinessAccountGate.AccessTokenEncrypted,
		"AccessTokenExpiresAt": wabaGate.WhatsAppBusinessAccountGate.AccessTokenExpiresAt,
		"BusinessID":           wabaGate.WhatsAppBusinessAccountGate.BusinessID,
	}

	return stmt, args
}
