package gate

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/webitel/im-providers-service/infra/auth"
	"github.com/webitel/im-providers-service/internal/whatsapp/client"
	"github.com/webitel/im-providers-service/internal/whatsapp/common"
	"github.com/webitel/im-providers-service/internal/whatsapp/messaging"
	"github.com/webitel/webitel-go-kit/pkg/errors"
	"github.com/webitel/webitel-go-kit/pkg/semconv"
	"golang.org/x/sync/errgroup"
)

type InternalContactResolver interface {
	Search(ctx context.Context, peer Peer) (Peer, error)
}

type WABAGateRepository interface {
	Save(ctx context.Context, wabaGate *Gate) (*Gate, error)
}

type gate struct {
	logger                  *slog.Logger
	wabaGateRepository      WABAGateRepository
	internalContactResolver InternalContactResolver
	encryptor               common.Encryptor
}

func newGate(logger *slog.Logger, wabaGateRepository WABAGateRepository, internalContactResolver InternalContactResolver, encryptor common.Encryptor) *gate {
	return &gate{
		logger:                  logger,
		wabaGateRepository:      wabaGateRepository,
		internalContactResolver: internalContactResolver,
		encryptor:               encryptor,
	}
}

// TODO:
// - add user claims via webitel authZ
// - add broker event for created whatsapp gate
// - add gate `access_token_expires_at` set after debug token request
func (gate *gate) Save(ctx context.Context, wabaGate *Gate) (*Gate, error) {
	log := gate.logger.With("operation", "whatsapp.gate.save")

	session, ok := auth.GetIdentityFromContext(ctx)
	if !ok {
		return nil, errors.Unauthenticated("not found session in context", errors.WithID("whatsapp.gate.save.session"))
	}

	wabaGate.DC = session.GetDomainID()

	if err := wabaGate.Validate(); err != nil {
		log.Warn("validating WhatsApp Business Account creating request", semconv.ErrorKey, err)
		return nil, err
	}

	if err := wabaGate.WhatsAppBusinessAccountGate.SetUpClient(); err != nil {
		log.Error("setupping WhatsApp request client", semconv.ErrorKey, err)
		return nil, err
	}

	if err := gate.performExternalWhatsAppAccountValidation(ctx, &wabaGate.WhatsAppBusinessAccountGate, wabaGate.Contact); err != nil {
		log.Error("performing external WhatsApp Business Account validation", semconv.ErrorKey, err, "phone_number_id", wabaGate.WhatsAppBusinessAccountGate.PhoneNumberID)
		return nil, err
	}

	encryptedAccessGate, err := wabaGate.WhatsAppBusinessAccountGate.PreSave(gate.encryptor)
	if err != nil {
		log.Error("performing pre-save access encrypting", semconv.ErrorKey, err, "phone_number_id", wabaGate.WhatsAppBusinessAccountGate.PhoneNumberID)
		return nil, err
	}
	wabaGate.WhatsAppBusinessAccountGate = encryptedAccessGate

	saved, err := gate.wabaGateRepository.Save(ctx, wabaGate)
	if err != nil {
		log.Error(
			"saving WhatsApp Business Account into database",
			semconv.ErrorKey, err,
			"meta_app_id", wabaGate.WhatsAppBusinessAccountGate.MetaAppID.String(),
			"phone_id", wabaGate.WhatsAppBusinessAccountGate.PhoneNumberID,
			"phone", wabaGate.WhatsAppBusinessAccountGate.PhoneNumber,
		)
		return nil, err
	}

	return saved, nil
}

func (gate *gate) performExternalWhatsAppAccountValidation(ctx context.Context, wabaGate *WhatsAppBusinessAccountGate, contact Peer) error {
	g, ctxGroup := errgroup.WithContext(ctx)

	g.Go(func() error { return gate.validateInternalBindingContact(ctxGroup, contact) })
	g.Go(func() error {
		return gate.validateWhatsAppAccount(ctxGroup, wabaGate.PhoneNumberID, wabaGate.GetClient())
	})
	g.Go(func() error { return gate.validateWhatsAppAccountToken(ctxGroup, wabaGate) })

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}

func (gate *gate) validateWhatsAppAccount(ctx context.Context, phoneNumberID string, requestClient client.RequestClient) error {
	req := requestClient.NewApiRequest(phoneNumberID, http.MethodGet)

	req.AddField(client.ApiRequestParamField{Name: "id"})
	req.AddField(client.ApiRequestParamField{Name: "display_phone_number"})
	req.AddField(client.ApiRequestParamField{Name: "verified_name"})

	response, err := req.ExecuteWithContext(ctx)
	if err != nil {
		return errors.Unauthenticated("validating WhatsApp phone", errors.WithCause(err), errors.WithID("gate.usecase.validate_whats_app_account"))
	}

	unmarshaledResponse, err := messaging.UnmarshalStatusResponse(response)
	if err != nil {
		return err
	}

	if unmarshaledResponse.Error != nil {
		return errors.Wrap(unmarshaledResponse.Error.ToGRPCError(), errors.WithID("gate.usecase.validate_whats_app_account"))
	}

	return nil
}

func (gate *gate) validateWhatsAppAccountToken(ctx context.Context, waba *WhatsAppBusinessAccountGate) error {
	requestClient := waba.GetClient()
	req := requestClient.NewApiRequest("debug_token", http.MethodGet)

	req.AddQueryParam("input_token", requestClient.AccessToken())

	resp, err := req.ExecuteWithContext(ctx)
	if err != nil {
		return errors.Unauthenticated("token is invalid", errors.WithCause(err))
	}

	unmarshaledResponse, err := messaging.UnmarshalStatusResponse(resp)
	if err != nil {
		return err
	}

	if unmarshaledResponse.Error != nil {
		return errors.Wrap(unmarshaledResponse.Error.ToGRPCError(), errors.WithID("gate.usecase.validate_whats_app_account"))
	}

	return nil
}

func (gate *gate) validateInternalBindingContact(ctx context.Context, internalPeer Peer) error {
	response, err := gate.internalContactResolver.Search(ctx, internalPeer)
	if err != nil {
		return errors.Internal("executing request to validate internal contact", errors.WithCause(err), errors.WithID("gate.usecase.validate_internal_binding_contact"))
	}

	if response == nil {
		return errors.NotFound("contact that wanted to be binded to gate not exists", errors.WithID("gate.usecase.validate_internal_binding_contact"))
	}

	return nil
}
