package instagram

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/webitel/webitel-go-kit/pkg/cache"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	igmodel "github.com/webitel/im-providers-service/internal/instagram/model"
)

type stubGraphAPI struct {
	sendTextFn           func(ctx context.Context, token, igsid, text string) (*sharedmodel.MessageResponse, error)
	sendHumanAgentTextFn func(ctx context.Context, token, igsid, text string) (*sharedmodel.MessageResponse, error)
}

func (s *stubGraphAPI) GetUserProfile(ctx context.Context, igsid, token string) (*UserProfile, error) {
	return nil, errors.New("not implemented") //nolint:nilnil
}

func (s *stubGraphAPI) ParseWebhook(data []byte) (*WebhookRequest, error) {
	return nil, errors.New("not implemented") //nolint:nilnil
}

func (s *stubGraphAPI) SendText(ctx context.Context, token, igsid, text string) (*sharedmodel.MessageResponse, error) {
	if s.sendTextFn != nil {
		return s.sendTextFn(ctx, token, igsid, text)
	}

	return nil, errors.New("not implemented") //nolint:nilnil
}

func (s *stubGraphAPI) SendMedia(ctx context.Context, token, igsid, mediaType, rawURL string) (*sharedmodel.MessageResponse, error) {
	return nil, errors.New("not implemented") //nolint:nilnil
}

func (s *stubGraphAPI) SendInteractive(ctx context.Context, token, igsid, body string, interactive *sharedmodel.Interactive) (*sharedmodel.MessageResponse, error) {
	return nil, errors.New("not implemented") //nolint:nilnil
}

func (s *stubGraphAPI) SendHumanAgentText(ctx context.Context, token, igsid, text string) (*sharedmodel.MessageResponse, error) {
	if s.sendHumanAgentTextFn != nil {
		return s.sendHumanAgentTextFn(ctx, token, igsid, text)
	}

	return nil, errors.New("not implemented") //nolint:nilnil
}

func (s *stubGraphAPI) SendTyping(ctx context.Context, token, igsid string, on bool) error {
	return nil
}

type stubGateStore struct {
	selectFn func(ctx context.Context, id string) (*igmodel.InstagramGate, error)
}

func (s *stubGateStore) Select(ctx context.Context, id string) (*igmodel.InstagramGate, error) {
	if s.selectFn != nil {
		return s.selectFn(ctx, id)
	}

	return nil, errors.New("not found")
}

func (s *stubGateStore) SelectByBusinessAccountAndURI(ctx context.Context, businessAccountID, uri string) (*igmodel.InstagramGate, error) {
	return nil, errors.New("not implemented") //nolint:nilnil
}

func (s *stubGateStore) Insert(ctx context.Context, domainID int64, gate *igmodel.InstagramGate) error {
	return nil
}

func (s *stubGateStore) Update(ctx context.Context, gate *igmodel.InstagramGate) error {
	return nil
}

func (s *stubGateStore) Delete(ctx context.Context, id string) error {
	return nil
}

func (s *stubGateStore) ListAll(ctx context.Context, offset, limit int32) ([]*igmodel.InstagramGate, error) {
	return nil, nil
}

func (s *stubGateStore) Unbind(ctx context.Context, gateID string) error {
	return nil
}

func outboundProvider(api graphAPI, gateStore *stubGateStore) *instagramProvider {
	igsidCache, _ := cache.New[string, string]().L1(cache.RistrettoConfig{MaxCost: 100, NumCounters: 1000}).Build()

	return &instagramProvider{
		logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		api:        api,
		repo:       gateStore,
		igsidCache: igsidCache,
		rdb:        nil,
	}
}

func TestSendText_TextTooLong(t *testing.T) {
	longText := string(make([]byte, 1001))

	api := &stubGraphAPI{}
	store := &stubGateStore{
		selectFn: func(ctx context.Context, id string) (*igmodel.InstagramGate, error) {
			return &igmodel.InstagramGate{
				ID:      id,
				IGToken: "token",
			}, nil
		},
	}
	p := outboundProvider(api, store)

	_, err := p.SendText(context.Background(), &sharedmodel.Message{
		GateID: "gate-1",
		Text:   longText,
		To:     sharedmodel.Peer{Sub: "12345"},
	})
	if err == nil {
		t.Fatal("expected error for text > 1000 chars")
	}

	if err.Error() != "instagram: message text exceeds 1000 characters" {
		t.Errorf("got unexpected error: %v", err)
	}
}

func TestSendText_OutsideMessageWindow_UsesHumanAgentTag(t *testing.T) {
	api := &stubGraphAPI{
		sendHumanAgentTextFn: func(ctx context.Context, token, igsid, text string) (*sharedmodel.MessageResponse, error) {
			if text != "Outside window" {
				t.Errorf("unexpected text: %s", text)
			}

			return &sharedmodel.MessageResponse{ID: "msg-2"}, nil
		},
	}
	store := &stubGateStore{
		selectFn: func(ctx context.Context, id string) (*igmodel.InstagramGate, error) {
			return &igmodel.InstagramGate{ID: id, IGToken: "token"}, nil
		},
	}
	p := outboundProvider(api, store)

	resp, err := p.SendText(context.Background(), &sharedmodel.Message{
		GateID: "gate-1",
		Text:   "Outside window",
		To:     sharedmodel.Peer{Sub: "12345"},
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if resp == nil || resp.ID != "msg-2" {
		t.Errorf("expected human agent response with ID msg-2, got %v", resp)
	}
}

func TestResolveIGSID_NumericIGSIDPassthrough(t *testing.T) {
	api := &stubGraphAPI{}
	store := &stubGateStore{}
	p := outboundProvider(api, store)

	igsid, err := p.resolveIGSID(context.Background(), &igmodel.InstagramGate{}, "12345")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if igsid != "12345" {
		t.Errorf("expected passthrough of numeric IGSID, got %s", igsid)
	}
}

func TestWithRecipient_StampsResponseMetadata(t *testing.T) {
	resp := &sharedmodel.MessageResponse{ID: "msg-1", MD: nil}
	igsid := "igsid-123"

	fn := withRecipient(resp, nil)

	result, err := fn(igsid)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.MD["recipient_id"] != igsid {
		t.Errorf("expected recipient_id in MD, got %v", result.MD)
	}
}

func TestWithRecipient_PreservesError(t *testing.T) {
	testErr := errors.New("send failed")

	fn := withRecipient(nil, testErr)
	_, err := fn("igsid-123")

	if !errors.Is(err, testErr) {
		t.Errorf("expected original error, got %v", err)
	}
}

func TestSendHumanAgentText_StampsRecipient(t *testing.T) {
	api := &stubGraphAPI{
		sendHumanAgentTextFn: func(ctx context.Context, token, igsid, text string) (*sharedmodel.MessageResponse, error) {
			return &sharedmodel.MessageResponse{ID: "msg-3"}, nil
		},
	}
	store := &stubGateStore{}
	p := outboundProvider(api, store)

	resp, err := p.sendHumanAgentText(context.Background(), "token-1", "igsid-456", "Hi there", "igsid-456")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if resp.MD["recipient_id"] != "igsid-456" {
		t.Errorf("expected recipient stamped in MD, got %v", resp.MD)
	}
}
