package instagram

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"testing"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
	fbmodel "github.com/webitel/im-providers-service/internal/facebook/model"
	fbstore "github.com/webitel/im-providers-service/internal/facebook/store"
	igmodel "github.com/webitel/im-providers-service/internal/instagram/model"
	"github.com/webitel/im-providers-service/internal/provider"
)

// ---------------------------------------------------------------------------
// Shared stubs (webhook-inbound tests)
// ---------------------------------------------------------------------------

// stubMetaAppRepo satisfies fbstore.MetaAppStore.
type stubMetaAppRepo struct {
	app *fbmodel.MetaApp
	err error
}

func (s *stubMetaAppRepo) Select(_ context.Context, _ string) (*fbmodel.MetaApp, error) {
	return s.app, s.err
}

func (s *stubMetaAppRepo) SelectByURI(_ context.Context, _ string) (*fbmodel.MetaApp, error) {
	return s.app, s.err
}

func (s *stubMetaAppRepo) Insert(_ context.Context, _ *fbmodel.MetaApp) error { return nil }
func (s *stubMetaAppRepo) Update(_ context.Context, _ *fbmodel.MetaApp) error { return nil }
func (s *stubMetaAppRepo) Delete(_ context.Context, _ string) error           { return nil }

var _ fbstore.MetaAppStore = (*stubMetaAppRepo)(nil)

// stubGateCache satisfies sharedstore.GateCache.
type stubGateCache struct {
	state   sharedstore.GateState
	present bool
}

func (s *stubGateCache) Set(_ string, state sharedstore.GateState) { s.state = state }
func (s *stubGateCache) Get(_ string) (sharedstore.GateState, bool) {
	return s.state, s.present
}

func (s *stubGateCache) Delete(_ string) {}

var _ sharedstore.GateCache = (*stubGateCache)(nil)

// stubInstagramRepo satisfies igstore.InstagramStore.
type stubInstagramRepo struct {
	selectFn                        func(ctx context.Context, id string) (*igmodel.InstagramGate, error)
	selectByBusinessAccountAndURIFn func(ctx context.Context, businessAccountID, uri string) (*igmodel.InstagramGate, error)
	insertCalled                    bool
	updateCalled                    bool
	deleteCalled                    bool
}

func (s *stubInstagramRepo) Select(ctx context.Context, id string) (*igmodel.InstagramGate, error) {
	if s.selectFn != nil {
		return s.selectFn(ctx, id)
	}

	return nil, errors.New("not found")
}

func (s *stubInstagramRepo) SelectByBusinessAccountAndURI(ctx context.Context, baid, uri string) (*igmodel.InstagramGate, error) {
	if s.selectByBusinessAccountAndURIFn != nil {
		return s.selectByBusinessAccountAndURIFn(ctx, baid, uri)
	}

	return nil, errors.New("not implemented")
}

func (s *stubInstagramRepo) Insert(_ context.Context, _ int64, _ *igmodel.InstagramGate) error {
	s.insertCalled = true

	return nil
}

func (s *stubInstagramRepo) Update(_ context.Context, _ *igmodel.InstagramGate) error {
	s.updateCalled = true

	return nil
}

func (s *stubInstagramRepo) Unbind(_ context.Context, _ string) error {
	s.deleteCalled = true

	return nil
}

func (s *stubInstagramRepo) ListAll(_ context.Context, _, _ int32) ([]*igmodel.InstagramGate, error) {
	return nil, nil //nolint:nilnil // stub — no result and no error is the expected empty state
}

// countingMessenger records Send* call counts and satisfies sharedsvc.Messenger.
type countingMessenger struct {
	sendTextCalls    int
	sendImageCalls   int
	sendDocCalls     int
	lastSendTextReq  *sharedmodel.SendTextRequest
	lastSendImageReq *sharedmodel.SendImageRequest
}

func (m *countingMessenger) SendText(_ context.Context, req *sharedmodel.SendTextRequest) (*sharedmodel.SendTextResponse, error) {
	m.sendTextCalls++
	m.lastSendTextReq = req

	return &sharedmodel.SendTextResponse{}, nil
}

func (m *countingMessenger) SendImage(_ context.Context, req *sharedmodel.SendImageRequest) (*sharedmodel.SendImageResponse, error) {
	m.sendImageCalls++
	m.lastSendImageReq = req

	return &sharedmodel.SendImageResponse{}, nil
}

func (m *countingMessenger) SendDocument(_ context.Context, _ *sharedmodel.SendDocumentRequest) (*sharedmodel.SendDocumentResponse, error) {
	m.sendDocCalls++

	return &sharedmodel.SendDocumentResponse{}, nil
}

func (m *countingMessenger) SendLocation(_ context.Context, _ *sharedmodel.SendLocationRequest) (*sharedmodel.SendResponse, error) {
	return nil, nil //nolint:nilnil // stub — caller ignores result
}

func (m *countingMessenger) SendContact(_ context.Context, _ *sharedmodel.SendContactRequest) (*sharedmodel.SendResponse, error) {
	return nil, nil //nolint:nilnil // stub — caller ignores result
}

func (m *countingMessenger) SendInteractiveCallback(_ context.Context, _ *sharedmodel.SendInteractiveCallbackRequest) error {
	return nil
}

func (m *countingMessenger) UpdateMessageDelivery(_ context.Context, _ *sharedmodel.MessageDeliveryReport) error {
	return nil
}

// minimalProvider builds a provider with only the fields needed for webhook/signature/verify tests.
func minimalProvider(metaRepo fbstore.MetaAppStore) *instagramProvider {
	return &instagramProvider{
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		metaAppRepo: metaRepo,
	}
}

// ctxWithURI injects a webhook URI into a context the same way the HTTP layer does.
func ctxWithURI(uri string) context.Context {
	return context.WithValue(context.Background(), provider.WebhookURIKey, uri)
}

// ---------------------------------------------------------------------------
// ParseWebhook / object filter
// ---------------------------------------------------------------------------

func TestParseWebhook_ObjectInstagram_Passes(t *testing.T) {
	payload := `{"object":"instagram","entry":[{"id":"biz-1","messaging":[]}]}`
	c := &apiClient{}

	req, err := c.ParseWebhook([]byte(payload))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := req.ParseWebhook(); got == nil {
		t.Fatal("expected non-nil result for object=instagram")
	}
}

func TestParseWebhook_ObjectPage_DropsPayload(t *testing.T) {
	payload := `{"object":"page","entry":[{"id":"page-1","messaging":[]}]}`
	c := &apiClient{}

	req, err := c.ParseWebhook([]byte(payload))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if got := req.ParseWebhook(); got != nil {
		t.Fatalf("expected nil for object=page, got %+v", got)
	}
}

func TestParseWebhook_ObjectEmpty_DropsPayload(t *testing.T) {
	payload := `{"object":"","entry":[]}`
	c := &apiClient{}

	req, _ := c.ParseWebhook([]byte(payload))
	if req.ParseWebhook() != nil {
		t.Fatal("expected nil for empty object field")
	}
}

// ---------------------------------------------------------------------------
// ValidateSignature
// ---------------------------------------------------------------------------

func makeSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)

	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestValidateSignature_CorrectHMAC_Passes(t *testing.T) {
	body := []byte(`{"object":"instagram"}`)
	secret := "test-secret"
	sig := makeSignature(secret, body)

	p := minimalProvider(&stubMetaAppRepo{app: &fbmodel.MetaApp{AppSecret: secret}})

	hdr := http.Header{}
	hdr.Set("X-Hub-Signature-256", sig)

	if err := p.ValidateSignature(ctxWithURI("/ig"), hdr, body); err != nil {
		t.Errorf("expected valid signature to pass, got: %v", err)
	}
}

func TestValidateSignature_WrongSecret_Fails(t *testing.T) {
	body := []byte(`{"object":"instagram"}`)
	sig := makeSignature("correct-secret", body)

	p := minimalProvider(&stubMetaAppRepo{app: &fbmodel.MetaApp{AppSecret: "wrong-secret"}})

	hdr := http.Header{}
	hdr.Set("X-Hub-Signature-256", sig)

	if err := p.ValidateSignature(ctxWithURI("/ig"), hdr, body); err == nil {
		t.Fatal("expected signature mismatch error")
	}
}

func TestValidateSignature_MissingHeader_Fails(t *testing.T) {
	p := minimalProvider(&stubMetaAppRepo{app: &fbmodel.MetaApp{AppSecret: "secret"}})

	if err := p.ValidateSignature(ctxWithURI("/ig"), http.Header{}, []byte("body")); err == nil {
		t.Fatal("expected error for missing header")
	}
}

func TestValidateSignature_BadPrefix_Fails(t *testing.T) {
	p := minimalProvider(&stubMetaAppRepo{app: &fbmodel.MetaApp{AppSecret: "secret"}})

	hdr := http.Header{}
	hdr.Set("X-Hub-Signature-256", "md5=abcdef")

	if err := p.ValidateSignature(ctxWithURI("/ig"), hdr, []byte("body")); err == nil {
		t.Fatal("expected error for non-sha256= prefix")
	}
}

func TestValidateSignature_AppLookupError_Fails(t *testing.T) {
	p := minimalProvider(&stubMetaAppRepo{err: errors.New("db error")})

	hdr := http.Header{}
	hdr.Set("X-Hub-Signature-256", "sha256=abc123")

	if err := p.ValidateSignature(ctxWithURI("/ig"), hdr, []byte("body")); err == nil {
		t.Fatal("expected error when app lookup fails")
	}
}

// ---------------------------------------------------------------------------
// Verify (hub challenge handshake)
// ---------------------------------------------------------------------------

func TestVerify_MatchingToken_ReturnsChallenge(t *testing.T) {
	p := minimalProvider(&stubMetaAppRepo{app: &fbmodel.MetaApp{VerifyToken: "secret-token"}})

	q := url.Values{}
	q.Set("hub.mode", "subscribe")
	q.Set("hub.verify_token", "secret-token")
	q.Set("hub.challenge", "challenge-xyz")

	ch, err := p.Verify(ctxWithURI("/ig"), q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ch != "challenge-xyz" {
		t.Errorf("expected challenge-xyz, got %q", ch)
	}
}

func TestVerify_TokenMismatch_Fails(t *testing.T) {
	p := minimalProvider(&stubMetaAppRepo{app: &fbmodel.MetaApp{VerifyToken: "expected"}})

	q := url.Values{}
	q.Set("hub.mode", "subscribe")
	q.Set("hub.verify_token", "wrong")
	q.Set("hub.challenge", "ch")

	if _, err := p.Verify(ctxWithURI("/ig"), q); err == nil {
		t.Fatal("expected error for token mismatch")
	}
}

func TestVerify_WrongMode_Fails(t *testing.T) {
	p := minimalProvider(&stubMetaAppRepo{app: &fbmodel.MetaApp{VerifyToken: "tok"}})

	q := url.Values{}
	q.Set("hub.mode", "unsubscribe")
	q.Set("hub.verify_token", "tok")
	q.Set("hub.challenge", "ch")

	if _, err := p.Verify(ctxWithURI("/ig"), q); err == nil {
		t.Fatal("expected error for mode != subscribe")
	}
}

func TestVerify_EmptyVerifyToken_AllowsAny(t *testing.T) {
	// When the app has no verify token configured, any caller token is accepted.
	p := minimalProvider(&stubMetaAppRepo{app: &fbmodel.MetaApp{VerifyToken: ""}})

	q := url.Values{}
	q.Set("hub.mode", "subscribe")
	q.Set("hub.verify_token", "anything")
	q.Set("hub.challenge", "ch")

	ch, err := p.Verify(ctxWithURI("/ig"), q)
	if err != nil {
		t.Fatalf("unexpected error for empty stored token: %v", err)
	}

	if ch != "ch" {
		t.Errorf("expected ch, got %q", ch)
	}
}

// ---------------------------------------------------------------------------
// resolveGate: disabled gate served from GateCache without DB hit
// ---------------------------------------------------------------------------

func TestResolveGate_DisabledFromCache_NoDB(t *testing.T) {
	gateCache := &stubGateCache{
		state:   sharedstore.GateState{GateID: "g-1", Enabled: false},
		present: true,
	}

	dbHit := false
	repo := &stubInstagramRepo{
		selectByBusinessAccountAndURIFn: func(_ context.Context, _, _ string) (*igmodel.InstagramGate, error) {
			dbHit = true

			return nil, errors.New("should not be called")
		},
	}

	p := &instagramProvider{
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		gateCache: gateCache,
		repo:      repo,
	}

	g, err := p.resolveGate(context.Background(), "/ig", "biz-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if g.Enabled {
		t.Error("expected disabled gate from cache")
	}

	if dbHit {
		t.Error("DB must not be hit when disabled gate is cached")
	}
}

func TestResolveGate_NotCached_HitsDB(t *testing.T) {
	gateCache := &stubGateCache{present: false}

	expectedGate := &igmodel.InstagramGate{
		ID:                "g-2",
		Enabled:           true,
		BusinessAccountID: "biz-2",
		Peer:              sharedmodel.Peer{Iss: "domain"},
	}

	dbHit := false
	repo := &stubInstagramRepo{
		selectByBusinessAccountAndURIFn: func(_ context.Context, baid, _ string) (*igmodel.InstagramGate, error) {
			dbHit = true

			if baid != "biz-2" {
				return nil, fmt.Errorf("unexpected baid: %s", baid)
			}

			return expectedGate, nil
		},
	}

	p := &instagramProvider{
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		gateCache: gateCache,
		repo:      repo,
	}

	g, err := p.resolveGate(context.Background(), "/ig", "biz-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !g.Enabled {
		t.Error("expected enabled gate from DB")
	}

	if !dbHit {
		t.Error("expected DB hit on cache miss")
	}

	// State must be written back to cache.
	if gateCache.state.GateID != "g-2" {
		t.Errorf("expected cache to be populated, got %+v", gateCache.state)
	}
}

// ---------------------------------------------------------------------------
// processMessage: is_echo skip
// ---------------------------------------------------------------------------

func TestProcessMessage_IsEcho_Skipped(t *testing.T) {
	messenger := &countingMessenger{}
	p := &instagramProvider{
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		messenger: messenger,
	}

	gate := &igmodel.InstagramGate{ID: "g-1", Enabled: true}

	err := p.processMessage(context.Background(), gate, Messaging{
		Sender: Actor{ID: "igsid-1"},
		Message: &InboundMessage{
			Mid:    "mid-echo",
			Text:   "hello",
			IsEcho: true,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if messenger.sendTextCalls > 0 {
		t.Error("is_echo message must not be forwarded to messenger")
	}
}

// ---------------------------------------------------------------------------
// routeMessage: peer invariants (From.Sub, To.Via, ExternalID) — WMSG-377
// ---------------------------------------------------------------------------

func TestRouteMessage_TextPeers_WMSG377(t *testing.T) {
	messenger := &countingMessenger{}
	gateID := "gate-abc"
	gate := &igmodel.InstagramGate{
		ID:      gateID,
		Enabled: true,
		Peer:    sharedmodel.Peer{Sub: "biz-account-sub", Iss: "domain-iss"},
	}

	p := &instagramProvider{
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		messenger: messenger,
	}

	igsid := "igsid-sender"
	mid := "mid-abc-123"

	p.routeMessage(context.Background(), gate, peerPair{
		from: sharedmodel.Peer{Sub: igsid, Iss: gate.Peer.Iss},
		to:   sharedmodel.Peer{Sub: gate.Peer.Sub, Iss: gate.Peer.Iss, Via: &gate.ID},
	}, &InboundMessage{Mid: mid, Text: "hello"})

	if messenger.sendTextCalls != 1 {
		t.Fatalf("expected 1 SendText call, got %d", messenger.sendTextCalls)
	}

	req := messenger.lastSendTextReq

	if req.From.Sub != igsid {
		t.Errorf("From.Sub: want %q, got %q", igsid, req.From.Sub)
	}

	if req.To.Via == nil || *req.To.Via != gateID {
		t.Errorf("To.Via: want %q, got %v", gateID, req.To.Via)
	}

	if req.ExternalID != mid {
		t.Errorf("ExternalID: want %q, got %q", mid, req.ExternalID)
	}
}

// ---------------------------------------------------------------------------
// handleAttachments: IG-only types are skipped without error
// ---------------------------------------------------------------------------

func TestHandleAttachments_SkipTypes_NoSendNoPanic(t *testing.T) {
	skipTypes := []string{
		AttachmentTypeStoryMention,
		AttachmentTypeIGReel,
		AttachmentTypeReel,
		AttachmentTypeShare,
	}

	messenger := &countingMessenger{}
	p := &instagramProvider{
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		messenger: messenger,
	}

	gate := &igmodel.InstagramGate{ID: "g-1"}
	peers := peerPair{
		from: sharedmodel.Peer{Sub: "igsid-1"},
		to:   sharedmodel.Peer{Sub: "biz", Via: &gate.ID},
	}

	for _, typ := range skipTypes {
		t.Run(typ, func(t *testing.T) {
			before := messenger.sendImageCalls + messenger.sendDocCalls

			p.handleAttachments(context.Background(), gate, peers, []Attachment{
				{Type: typ, Payload: AttachmentPayload{URL: "https://example.com/media"}},
			}, "mid-1", "")

			after := messenger.sendImageCalls + messenger.sendDocCalls

			if after != before {
				t.Errorf("attachment type %q must be skipped — got unexpected Send call", typ)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// sendConversationStartedTemplate: nil templates → no-op
// ---------------------------------------------------------------------------

func TestSendConversationStartedTemplate_NilTemplates_NoOp(t *testing.T) {
	p := &instagramProvider{
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		templates: nil,
	}

	// Must not panic.
	p.sendConversationStartedTemplate(context.Background(), &igmodel.InstagramGate{ID: "g-1"}, "igsid-1", &UserProfile{Name: "Alice"})
}

// ---------------------------------------------------------------------------
// AllMessages: flattening across multiple entries
// ---------------------------------------------------------------------------

func TestAllMessages_MultipleEntries(t *testing.T) {
	req := &WebhookRequest{
		Object: "instagram",
		Entry: []Entry{
			{ID: "e1", Messaging: []Messaging{{Sender: Actor{ID: "a"}}, {Sender: Actor{ID: "b"}}}},
			{ID: "e2", Messaging: []Messaging{{Sender: Actor{ID: "c"}}}},
		},
	}

	msgs := req.AllMessages()
	if len(msgs) != 3 {
		t.Errorf("expected 3 messages, got %d", len(msgs))
	}
}

func TestAllMessages_Empty(t *testing.T) {
	req := &WebhookRequest{Object: "instagram"}

	if msgs := req.AllMessages(); len(msgs) != 0 {
		t.Errorf("expected 0 messages, got %d", len(msgs))
	}
}

// ---------------------------------------------------------------------------
// isTokenInvalidError
// ---------------------------------------------------------------------------

func tokenErrorBody(code int) []byte {
	b, _ := json.Marshal(map[string]any{
		"error": map[string]any{"code": code, "message": "invalid token"},
	})

	return b
}

func TestIsTokenInvalidError_Code190_True(t *testing.T) {
	if !isTokenInvalidError(tokenErrorBody(190)) {
		t.Error("code 190 must be detected as token-invalid")
	}
}

func TestIsTokenInvalidError_OtherCode_False(t *testing.T) {
	for _, code := range []int{100, 200, 0, 10, 4} {
		if isTokenInvalidError(tokenErrorBody(code)) {
			t.Errorf("code %d must NOT be detected as token-invalid", code)
		}
	}
}

func TestIsTokenInvalidError_MalformedJSON_False(t *testing.T) {
	if isTokenInvalidError([]byte("not json")) {
		t.Error("malformed JSON must return false")
	}
}

// ---------------------------------------------------------------------------
// Payload builders: shape validation
// ---------------------------------------------------------------------------

func TestNewTextPayload_Shape(t *testing.T) {
	p := newTextPayload("igsid-1", "hello")

	if p.Type != "RESPONSE" {
		t.Errorf("messaging_type: want RESPONSE, got %s", p.Type)
	}

	if p.Recipient.ID != "igsid-1" {
		t.Errorf("recipient.id: want igsid-1, got %s", p.Recipient.ID)
	}

	if p.Message.Text != "hello" {
		t.Errorf("message.text: want hello, got %s", p.Message.Text)
	}
}

func TestNewHumanAgentTextPayload_Shape(t *testing.T) {
	p := NewHumanAgentTextPayload("igsid-2", "agent msg")

	if p.Type != "MESSAGE_TAG" {
		t.Errorf("messaging_type: want MESSAGE_TAG, got %s", p.Type)
	}

	if p.Recipient.ID != "igsid-2" {
		t.Errorf("recipient.id: want igsid-2, got %s", p.Recipient.ID)
	}
}

// ---------------------------------------------------------------------------
// buildInteractiveMessage
// ---------------------------------------------------------------------------

func TestBuildInteractiveMessage_NilInteractive_Error(t *testing.T) {
	if _, err := buildInteractiveMessage("body", nil); err == nil {
		t.Error("expected error for nil interactive")
	}
}

func TestBuildInteractiveMessage_NoButtons_Error(t *testing.T) {
	_, err := buildInteractiveMessage("body", &sharedmodel.Interactive{
		Markup: &sharedmodel.KeyboardMarkup{},
	})
	if err == nil {
		t.Error("expected error when no buttons are provided")
	}
}

func TestBuildInteractiveMessage_CallbackButtons_Mapped(t *testing.T) {
	interactive := &sharedmodel.Interactive{
		Markup: &sharedmodel.KeyboardMarkup{
			Rows: []sharedmodel.KeyboardRow{
				{Buttons: []sharedmodel.KeyboardButton{
					{Label: "Yes", Callback: &sharedmodel.KeyboardButtonCallback{Data: "yes"}},
					{Label: "No", Callback: &sharedmodel.KeyboardButtonCallback{Data: "no"}},
				}},
			},
		},
	}

	msg, err := buildInteractiveMessage("Pick one", interactive)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg.Attachment == nil {
		t.Fatal("expected attachment (button template)")
	}
}

func TestBuildInteractiveMessage_MaxThreeButtons(t *testing.T) {
	// Only 3 of 5 buttons should be included — Instagram template limit.
	rows := []sharedmodel.KeyboardRow{{}}

	for i := range 5 {
		rows[0].Buttons = append(rows[0].Buttons, sharedmodel.KeyboardButton{
			Label:    fmt.Sprintf("Btn%d", i),
			Callback: &sharedmodel.KeyboardButtonCallback{Data: fmt.Sprintf("d%d", i)},
		})
	}

	interactive := &sharedmodel.Interactive{Markup: &sharedmodel.KeyboardMarkup{Rows: rows}}

	msg, err := buildInteractiveMessage("body", interactive)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	payload, ok := msg.Attachment.Payload.(igButtonTemplatePayload)
	if !ok {
		t.Fatalf("expected igButtonTemplatePayload, got %T", msg.Attachment.Payload)
	}

	if len(payload.Buttons) != 3 {
		t.Errorf("expected exactly 3 buttons (IG limit), got %d", len(payload.Buttons))
	}
}

// ---------------------------------------------------------------------------
// attachmentFileName
// ---------------------------------------------------------------------------

func TestAttachmentFileName_UsesTitle(t *testing.T) {
	a := Attachment{Type: "image", Payload: AttachmentPayload{Title: "photo.jpg"}}

	if got := attachmentFileName(a); got != "photo.jpg" {
		t.Errorf("want photo.jpg, got %s", got)
	}
}

func TestAttachmentFileName_FallsBackToName(t *testing.T) {
	a := Attachment{Type: "image", Payload: AttachmentPayload{Name: "img.png"}}

	if got := attachmentFileName(a); got != "img.png" {
		t.Errorf("want img.png, got %s", got)
	}
}

func TestAttachmentFileName_GeneratesForKnownType(t *testing.T) {
	a := Attachment{Type: "video"}

	got := attachmentFileName(a)
	if len(got) == 0 {
		t.Error("expected generated filename")
	}
}

// ---------------------------------------------------------------------------
// readInt64: boundary / invalid
// ---------------------------------------------------------------------------

func TestReadInt64_ValidPositive(t *testing.T) {
	var v int64

	got, err := readInt64("12345", &v)
	if err != nil || got != 12345 {
		t.Errorf("got %d, err %v", got, err)
	}
}

func TestReadInt64_Zero(t *testing.T) {
	var v int64

	got, err := readInt64("0", &v)
	if err != nil || got != 0 {
		t.Errorf("got %d, err %v", got, err)
	}
}

func TestReadInt64_InvalidChar_Error(t *testing.T) {
	var v int64

	if _, err := readInt64("123x5", &v); err == nil {
		t.Error("expected error for non-numeric character")
	}
}

func TestReadInt64_NegativeNotSupported(t *testing.T) {
	var v int64

	if _, err := readInt64("-1", &v); err == nil {
		t.Error("expected error: minus sign is not a digit")
	}
}

// ---------------------------------------------------------------------------
// HandleWebhook: object=page → early nil return (no gate lookup)
// ---------------------------------------------------------------------------

// filteringGraphAPI parses JSON and applies the object==instagram filter
// the same way apiClient.ParseWebhook does.
type filteringGraphAPI struct {
	stubGraphAPI
}

func (f *filteringGraphAPI) ParseWebhook(data []byte) (*WebhookRequest, error) {
	var r WebhookRequest

	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}

	// Mirror apiClient: return nil when object != "instagram".
	return r.ParseWebhook(), nil
}

func TestHandleWebhook_ObjectPage_NilReturn_NoDownstream(t *testing.T) {
	gateHit := false

	p := &instagramProvider{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		api:    &filteringGraphAPI{},
		repo: &stubInstagramRepo{
			selectByBusinessAccountAndURIFn: func(_ context.Context, _, _ string) (*igmodel.InstagramGate, error) {
				gateHit = true

				return nil, errors.New("should not be called")
			},
		},
		gateCache: &stubGateCache{},
	}

	payload := `{"object":"page","entry":[{"id":"page-1","messaging":[{"sender":{"id":"u1"},"message":{"mid":"m1","text":"hi"}}]}]}`

	if err := p.HandleWebhook(context.Background(), []byte(payload)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gateHit {
		t.Error("gate must not be resolved for object=page")
	}
}
