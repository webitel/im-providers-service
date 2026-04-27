package gate

import (
	"context"

	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	"github.com/webitel/im-providers-service/internal/whatsapp/common"
	"github.com/webitel/webitel-go-kit/pkg/errors"
)

type GateEditor interface {
	Save(ctx context.Context, wabaGate *Gate) (*Gate, error)
}

type whatsAppBusinessAccountServer struct {
	impb.UnimplementedWhatsAppServiceServer

	editor GateEditor
}

func newWhatsAppBusinessAccountServer(editor GateEditor) *whatsAppBusinessAccountServer {
	return &whatsAppBusinessAccountServer{editor: editor}
}

func (server *whatsAppBusinessAccountServer) CreateWhatsAppGate(ctx context.Context, in *impb.CreateGateRequest) (*impb.GateResponse, error) {
	var whatsAppBusinessAccountGate WhatsAppBusinessAccountGate

	switch g := in.GetGate().(type) {
	case *impb.CreateGateRequest_Waba:
		whatsAppBusinessAccountGate = WhatsAppBusinessAccountGate{
			MetaAppID:            common.SafeConvertStringToUUID(g.Waba.GetMetaAppId()),
			PhoneNumber:          g.Waba.GetPhoneNumber(),
			PhoneNumberID:        g.Waba.GetPhoneNumberId(),
			AccessToken:          g.Waba.GetAccessToken(),
			AccessTokenExpiresAt: nil, //TODO
			BusinessID:           g.Waba.GetBusinessId(),
		}
	default:
		return nil, errors.InvalidArgument("unexpected gate type in create WhatsApp gate request", errors.WithID("gate.server.create_whatsapp_gate"))
	}

	gate := Gate{
		Name:                        in.GetName(),
		Type:                        WhatsAppGateType,
		Enabled:                     in.GetEnabled(),
		WhatsAppBusinessAccountGate: whatsAppBusinessAccountGate,
		Contact: &common.Contact{
			Iss: in.GetBot().GetIss(),
			Sub: in.GetBot().GetSub(),
		},
	}

	savedGate, err := server.editor.Save(ctx, &gate)
	if err != nil {
		return nil, err
	}

	response := impb.GateResponse{
		Id:        savedGate.ID.String(),
		Name:      savedGate.Name,
		Type:      savedGate.Type,
		Enabled:   savedGate.Enabled,
		CreatedAt: savedGate.CreatedAtUnixUTCMilli(),
		CreatedBy: savedGate.CreatedBy,
		UpdatedAt: savedGate.UpdatedAtUnixUTCMilli(),
		UpdatedBy: savedGate.UpdatedBy,
		Gate: &impb.GateResponse_Waba{
			Waba: &impb.WhatsAppBusinessAccount{
				MetaAppId:            savedGate.WhatsAppBusinessAccountGate.MetaAppID.String(),
				PhoneNumber:          savedGate.WhatsAppBusinessAccountGate.PhoneNumber,
				PhoneNumberId:        savedGate.WhatsAppBusinessAccountGate.PhoneNumberID,
				AccessToken:          string(savedGate.WhatsAppBusinessAccountGate.AccessTokenEncrypted),
				AccessTokenExpiresAt: nil,
				BusinessId:           savedGate.WhatsAppBusinessAccountGate.BusinessID,
			},
		},
		Bot: &impb.Peer{
			Sub: savedGate.Contact.Sub,
			Iss: savedGate.Contact.Iss,
		},
	}

	return &response, nil
}

func (server *whatsAppBusinessAccountServer) GetWhatsAppGate(ctx context.Context, in *impb.ProviderGetWhatsAppGateRequest) (*impb.ProviderGetWhatsAppGateResponse, error)
func (server *whatsAppBusinessAccountServer) UpdateWhatsAppGate(ctx context.Context, in *impb.ProviderUpdateWhatsAppGateRequest) (*impb.ProviderUpdateWhatsAppGateResponse, error)
func (server *whatsAppBusinessAccountServer) DeleteWhatsAppGate(ctx context.Context, in *impb.ProviderDeleteWhatsAppGateRequest) (*impb.ProviderDeleteWhatsAppGateResponse, error)
