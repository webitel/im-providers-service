package custom

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
	custommodel "github.com/webitel/im-providers-service/internal/custom/model"
	customstore "github.com/webitel/im-providers-service/internal/custom/store"
	"github.com/webitel/im-providers-service/internal/provider"
)

const (
	testURI    = "3f1c8a"
	testSecret = "s3cr3t"
	testGateID = "11111111-1111-1111-1111-111111111111"
)

var noopLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

// --- stubs ---

var _ customstore.CustomStore = (*stubStore)(nil)

type stubStore struct {
	gate *custommodel.CustomGate

	chatsByRecipient map[string]*custommodel.Chat
	chatsByID        map[string]*custommodel.Chat

	upserted []upsertedChat
}

type upsertedChat struct {
	gateID string
	chatID string
	sub    string
}

func (s *stubStore) Insert(context.Context, int64, *custommodel.CustomGate) error { return nil }

func (s *stubStore) Select(_ context.Context, _ string) (*custommodel.CustomGate, error) {
	if s.gate == nil {
		return nil, sharedstore.ErrNotFound
	}

	return s.gate, nil
}

func (s *stubStore) SelectByURI(_ context.Context, uri string) (*custommodel.CustomGate, error) {
	if s.gate == nil || s.gate.WebhookURI != uri {
		return nil, sharedstore.ErrNotFound
	}

	return s.gate, nil
}

func (s *stubStore) Update(context.Context, *custommodel.CustomGate) error { return nil }
func (s *stubStore) Unbind(context.Context, string) error                  { return nil }

func (s *stubStore) UpsertChat(_ context.Context, gateID, chatID, externalSub string) error {
	s.upserted = append(s.upserted, upsertedChat{gateID: gateID, chatID: chatID, sub: externalSub})

	return nil
}

func (s *stubStore) ChatByRecipient(_ context.Context, _, externalSub string) (*custommodel.Chat, error) {
	if chat, ok := s.chatsByRecipient[externalSub]; ok {
		return chat, nil
	}

	return nil, sharedstore.ErrNotFound
}

func (s *stubStore) ChatByID(_ context.Context, _, chatID string) (*custommodel.Chat, error) {
	if chat, ok := s.chatsByID[chatID]; ok {
		return chat, nil
	}

	return nil, sharedstore.ErrNotFound
}

type stubGateCache struct{}

func (stubGateCache) Set(string, sharedstore.GateState) {}
func (stubGateCache) Get(string) (sharedstore.GateState, bool) {
	return sharedstore.GateState{}, false
}
func (stubGateCache) Delete(string) {}

// knownUserCache reports every user as already synced, which keeps the
// contact-creation calls out of the inbound tests.
type knownUserCache struct{}

func (knownUserCache) IsKnown(context.Context, *sharedmodel.ExternalUser) (bool, error) {
	return true, nil
}
func (knownUserCache) MarkKnown(context.Context, *sharedmodel.ExternalUser) error { return nil }

type recordedMessenger struct {
	texts     []*sharedmodel.SendTextRequest
	images    []*sharedmodel.SendImageRequest
	documents []*sharedmodel.SendDocumentRequest
	locations []*sharedmodel.SendLocationRequest
	contacts  []*sharedmodel.SendContactRequest
	callbacks []*sharedmodel.SendInteractiveCallbackRequest

	callbackErr error
}

func (m *recordedMessenger) SendText(_ context.Context, in *sharedmodel.SendTextRequest) (*sharedmodel.SendTextResponse, error) {
	m.texts = append(m.texts, in)

	return &sharedmodel.SendTextResponse{}, nil
}

func (m *recordedMessenger) SendImage(_ context.Context, in *sharedmodel.SendImageRequest) (*sharedmodel.SendImageResponse, error) {
	m.images = append(m.images, in)

	return &sharedmodel.SendImageResponse{}, nil
}

func (m *recordedMessenger) SendDocument(_ context.Context, in *sharedmodel.SendDocumentRequest) (*sharedmodel.SendDocumentResponse, error) {
	m.documents = append(m.documents, in)

	return &sharedmodel.SendDocumentResponse{}, nil
}

func (m *recordedMessenger) SendLocation(_ context.Context, in *sharedmodel.SendLocationRequest) (*sharedmodel.SendResponse, error) {
	m.locations = append(m.locations, in)

	return &sharedmodel.SendResponse{}, nil
}

func (m *recordedMessenger) SendContact(_ context.Context, in *sharedmodel.SendContactRequest) (*sharedmodel.SendResponse, error) {
	m.contacts = append(m.contacts, in)

	return &sharedmodel.SendResponse{}, nil
}

func (m *recordedMessenger) SendInteractiveCallback(_ context.Context, in *sharedmodel.SendInteractiveCallbackRequest) error {
	m.callbacks = append(m.callbacks, in)

	return m.callbackErr
}

func (m *recordedMessenger) UpdateMessageDelivery(context.Context, *sharedmodel.MessageDeliveryReport) error {
	return nil
}

// --- helpers ---

func testGate() *custommodel.CustomGate {
	return &custommodel.CustomGate{
		ID:               testGateID,
		DomainID:         7,
		Name:             "Partner middleware",
		Peer:             sharedmodel.Peer{Sub: "bot-sub", Iss: "schema"},
		CallbackURL:      "https://partner.example.org/hook",
		AppSecret:        testSecret,
		WebhookURI:       testURI,
		RequestTimeoutMS: custommodel.DefaultRequestTimeoutMS,
		RetryAttempts:    custommodel.DefaultRetryAttempts,
		Enabled:          true,
	}
}

func newTestProvider(t *testing.T, store *stubStore, messenger *recordedMessenger) *customProvider {
	t.Helper()

	p, err := New(
		messenger,
		noopLogger,
		stubGateCache{},
		knownUserCache{},
		store,
		nil,
		nil,
		nil,
		nil,
		newClient(noopLogger),
		newFetcher(),
		nil,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	return p.(*customProvider)
}

func webhookCtx() context.Context {
	return context.WithValue(context.Background(), provider.WebhookURIKey, testURI)
}

func signature(body, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))

	return hex.EncodeToString(mac.Sum(nil))
}

func signedHeader(body, secret string) http.Header {
	h := http.Header{}
	h.Set(signHeader, signature(body, secret))

	return h
}

// --- authentication ---

func TestValidateSignature_AcceptsMatchingSignature(t *testing.T) {
	p := newTestProvider(t, &stubStore{gate: testGate()}, &recordedMessenger{})
	body := `{"message":{"id":"e1"}}`

	if err := p.ValidateSignature(webhookCtx(), signedHeader(body, testSecret), []byte(body)); err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}
}

func TestValidateSignature_RejectsWrongSecret(t *testing.T) {
	p := newTestProvider(t, &stubStore{gate: testGate()}, &recordedMessenger{})
	body := `{"message":{"id":"e1"}}`

	err := p.ValidateSignature(webhookCtx(), signedHeader(body, "other-secret"), []byte(body))
	if !errors.Is(err, custommodel.ErrSignatureInvalid) {
		t.Fatalf("want ErrSignatureInvalid, got %v", err)
	}
}

func TestValidateSignature_RejectsTamperedBody(t *testing.T) {
	p := newTestProvider(t, &stubStore{gate: testGate()}, &recordedMessenger{})
	body := `{"message":{"id":"e1"}}`
	header := signedHeader(body, testSecret)

	err := p.ValidateSignature(webhookCtx(), header, []byte(`{"message":{"id":"e2"}}`))
	if !errors.Is(err, custommodel.ErrSignatureInvalid) {
		t.Fatalf("want ErrSignatureInvalid, got %v", err)
	}
}

func TestValidateSignature_RejectsMissingHeader(t *testing.T) {
	p := newTestProvider(t, &stubStore{gate: testGate()}, &recordedMessenger{})

	err := p.ValidateSignature(webhookCtx(), http.Header{}, []byte(`{}`))
	if !errors.Is(err, custommodel.ErrSignatureInvalid) {
		t.Fatalf("want ErrSignatureInvalid, got %v", err)
	}
}

func TestValidateSignature_RejectsUnknownURI(t *testing.T) {
	p := newTestProvider(t, &stubStore{gate: testGate()}, &recordedMessenger{})
	body := `{}`

	ctx := context.WithValue(context.Background(), provider.WebhookURIKey, "not-our-uri")
	if err := p.ValidateSignature(ctx, signedHeader(body, testSecret), []byte(body)); err == nil {
		t.Fatal("unknown uri accepted")
	}
}

func TestValidateSignature_RejectsDisabledGate(t *testing.T) {
	gate := testGate()
	gate.Enabled = false

	p := newTestProvider(t, &stubStore{gate: gate}, &recordedMessenger{})
	body := `{}`

	if err := p.ValidateSignature(webhookCtx(), signedHeader(body, testSecret), []byte(body)); err == nil {
		t.Fatal("disabled gate accepted")
	}
}

func TestValidateSignature_EnforcesAllowedIPs(t *testing.T) {
	gate := testGate()
	gate.AllowedIPs = []string{"203.0.113.0/24"}

	p := newTestProvider(t, &stubStore{gate: gate}, &recordedMessenger{})
	body := `{}`

	allowed := signedHeader(body, testSecret)
	allowed.Set("X-Real-IP", "203.0.113.10")

	if err := p.ValidateSignature(webhookCtx(), allowed, []byte(body)); err != nil {
		t.Fatalf("allowlisted address rejected: %v", err)
	}

	denied := signedHeader(body, testSecret)
	denied.Set("X-Real-IP", "198.51.100.4")

	err := p.ValidateSignature(webhookCtx(), denied, []byte(body))
	if !errors.Is(err, custommodel.ErrSourceNotAllowed) {
		t.Fatalf("want ErrSourceNotAllowed, got %v", err)
	}
}

func TestValidateSignature_AllowsAnySourceWhenAllowlistEmpty(t *testing.T) {
	p := newTestProvider(t, &stubStore{gate: testGate()}, &recordedMessenger{})
	body := `{}`

	header := signedHeader(body, testSecret)
	header.Set("X-Real-IP", "198.51.100.4")

	if err := p.ValidateSignature(webhookCtx(), header, []byte(body)); err != nil {
		t.Fatalf("empty allowlist rejected a caller: %v", err)
	}
}

// --- inbound content ---

func TestHandleWebhook_TextCreatesMessageAndRecordsChat(t *testing.T) {
	store := &stubStore{gate: testGate()}
	messenger := &recordedMessenger{}
	p := newTestProvider(t, store, messenger)

	body := `{"message":{"id":"e1","chatId":"c1","sender":{"id":"u1","type":"viber","name":"John Doe"},"date":1710348821689,"text":"Hi!","metadata":{"order_id":"42"}}}`

	if err := p.HandleWebhook(webhookCtx(), []byte(body)); err != nil {
		t.Fatalf("HandleWebhook: %v", err)
	}

	if len(messenger.texts) != 1 {
		t.Fatalf("want 1 text message, got %d", len(messenger.texts))
	}

	sent := messenger.texts[0]
	if sent.Body != "Hi!" {
		t.Errorf("body = %q, want %q", sent.Body, "Hi!")
	}

	if sent.ExternalID != "e1" {
		t.Errorf("external id = %q, want e1", sent.ExternalID)
	}

	if sent.From.Sub != "viber|u1" {
		t.Errorf("sender sub = %q, want viber|u1", sent.From.Sub)
	}

	if sent.To.Sub != "bot-sub" {
		t.Errorf("recipient sub = %q, want bot-sub", sent.To.Sub)
	}

	if sent.Variables["order_id"] != "42" {
		t.Errorf("order_id variable = %q, want 42", sent.Variables["order_id"])
	}

	if sent.Variables[sourceVariable] != "viber" {
		t.Errorf("source variable = %q, want viber", sent.Variables[sourceVariable])
	}

	if len(store.upserted) != 1 || store.upserted[0].chatID != "c1" || store.upserted[0].sub != "viber|u1" {
		t.Fatalf("chat mapping not recorded: %+v", store.upserted)
	}
}

func TestHandleWebhook_VariablesOnlyOnTheMessageThatOpensTheChat(t *testing.T) {
	store := &stubStore{
		gate:      testGate(),
		chatsByID: map[string]*custommodel.Chat{"c1": {GateID: testGateID, ChatID: "c1", ExternalSub: "viber|u1"}},
	}
	messenger := &recordedMessenger{}
	p := newTestProvider(t, store, messenger)

	body := `{"message":{"id":"e2","chatId":"c1","sender":{"id":"u1","type":"viber"},"text":"more","metadata":{"order_id":"42"}}}`

	if err := p.HandleWebhook(webhookCtx(), []byte(body)); err != nil {
		t.Fatalf("HandleWebhook: %v", err)
	}

	if got := messenger.texts[0].Variables; got != nil {
		t.Errorf("variables on a later message = %v, want none", got)
	}
}

func TestHandleWebhook_LocationAndContact(t *testing.T) {
	store := &stubStore{gate: testGate()}
	messenger := &recordedMessenger{}
	p := newTestProvider(t, store, messenger)

	location := `{"message":{"id":"e1","chatId":"c1","sender":{"id":"u1"},"location":{"lat":50.45,"lon":30.52,"title":"Office","address":"Khreshchatyk 1"}}}`
	if err := p.HandleWebhook(webhookCtx(), []byte(location)); err != nil {
		t.Fatalf("location: %v", err)
	}

	contact := `{"message":{"id":"e2","chatId":"c1","sender":{"id":"u1"},"contact":{"name":"Jane","phone":"+380000000000","email":"jane@example.org"}}}`
	if err := p.HandleWebhook(webhookCtx(), []byte(contact)); err != nil {
		t.Fatalf("contact: %v", err)
	}

	if len(messenger.locations) != 1 {
		t.Fatalf("want 1 location, got %d", len(messenger.locations))
	}

	if got := messenger.locations[0]; got.Latitude != 50.45 || got.Longitude != 30.52 {
		t.Errorf("coordinates = %v,%v", got.Latitude, got.Longitude)
	}

	if got := messenger.locations[0].Name; got == nil || *got != "Office" {
		t.Errorf("location name = %v, want Office", got)
	}

	if messenger.locations[0].ExternalID != "e1" {
		t.Errorf("location external id = %q, want e1", messenger.locations[0].ExternalID)
	}

	if len(messenger.contacts) != 1 {
		t.Fatalf("want 1 contact, got %d", len(messenger.contacts))
	}

	if got := messenger.contacts[0].PhoneNumber; got == nil || *got != "+380000000000" {
		t.Errorf("contact phone = %v", got)
	}

	if messenger.contacts[0].ExternalID != "e2" {
		t.Errorf("contact external id = %q, want e2", messenger.contacts[0].ExternalID)
	}
}

func TestHandleWebhook_MenuCallbackBecomesInteractiveCallback(t *testing.T) {
	store := &stubStore{gate: testGate()}
	messenger := &recordedMessenger{}
	p := newTestProvider(t, store, messenger)

	body := `{"message":{"id":"e1","chatId":"c1","sender":{"id":"u1"},"callback":{"code":"rate_5","messageId":"22222222-2222-2222-2222-222222222222","data":"5"}}}`

	if err := p.HandleWebhook(webhookCtx(), []byte(body)); err != nil {
		t.Fatalf("HandleWebhook: %v", err)
	}

	if len(messenger.callbacks) != 1 {
		t.Fatalf("want 1 callback, got %d", len(messenger.callbacks))
	}

	cb := messenger.callbacks[0]
	if cb.InReplyTo != "22222222-2222-2222-2222-222222222222" {
		t.Errorf("in_reply_to = %q", cb.InReplyTo)
	}

	if cb.ButtonCode != "rate_5" || cb.CallbackData != "5" {
		t.Errorf("button = %q data = %q", cb.ButtonCode, cb.CallbackData)
	}

	if len(messenger.texts) != 0 {
		t.Errorf("callback should not also be forwarded as text")
	}
}

// A button press we cannot correlate must still reach the operator: the
// alternative is silently losing the customer's answer.
func TestHandleWebhook_MenuCallbackFallsBackToText(t *testing.T) {
	store := &stubStore{gate: testGate()}
	messenger := &recordedMessenger{callbackErr: errors.New("in_reply_to must be a valid message id")}
	p := newTestProvider(t, store, messenger)

	body := `{"message":{"id":"e1","chatId":"c1","sender":{"id":"u1"},"callback":{"code":"rate_5","messageId":"not-a-uuid","data":"5"}}}`

	if err := p.HandleWebhook(webhookCtx(), []byte(body)); err != nil {
		t.Fatalf("HandleWebhook: %v", err)
	}

	if len(messenger.texts) != 1 {
		t.Fatalf("want 1 fallback text, got %d", len(messenger.texts))
	}

	if messenger.texts[0].Body != "5" {
		t.Errorf("fallback body = %q, want 5", messenger.texts[0].Body)
	}
}

func TestHandleWebhook_RejectsPayloadWithoutEvent(t *testing.T) {
	p := newTestProvider(t, &stubStore{gate: testGate()}, &recordedMessenger{})

	if err := p.HandleWebhook(webhookCtx(), []byte(`{}`)); err == nil {
		t.Fatal("empty envelope accepted")
	}
}

func TestHandleWebhook_RejectsMessageWithoutChatID(t *testing.T) {
	p := newTestProvider(t, &stubStore{gate: testGate()}, &recordedMessenger{})

	body := `{"message":{"id":"e1","sender":{"id":"u1"},"text":"hi"}}`
	if err := p.HandleWebhook(webhookCtx(), []byte(body)); err == nil {
		t.Fatal("message without chatId accepted")
	}
}

func TestHandleWebhook_RejectsUnparsablePayload(t *testing.T) {
	p := newTestProvider(t, &stubStore{gate: testGate()}, &recordedMessenger{})

	if err := p.HandleWebhook(webhookCtx(), []byte(`{"message":`)); err == nil {
		t.Fatal("malformed json accepted")
	}
}

// --- response body ---

func TestWebhookResponse(t *testing.T) {
	p := newTestProvider(t, &stubStore{gate: testGate()}, &recordedMessenger{})

	contentType, body := p.WebhookResponse(nil)
	if contentType != "application/json" {
		t.Errorf("content type = %q", contentType)
	}

	if string(body) != `{"success":true}` {
		t.Errorf("success body = %s", body)
	}

	_, body = p.WebhookResponse(errors.New("boom"))
	if string(body) != `{"success":false,"error":"boom"}` {
		t.Errorf("failure body = %s", body)
	}
}

func TestCapabilities(t *testing.T) {
	p := newTestProvider(t, &stubStore{gate: testGate()}, &recordedMessenger{})

	caps := p.Capabilities()
	if !caps.SupportsDelivered || !caps.SupportsRead || !caps.SupportsFailed {
		t.Errorf("delivery capabilities not advertised: %+v", caps)
	}

	if caps.SupportsTyping {
		t.Error("typing advertised, but the contract has no typing event")
	}
}

func TestType_MatchesRegistryKey(t *testing.T) {
	p := newTestProvider(t, &stubStore{gate: testGate()}, &recordedMessenger{})

	// The registry resolves an outbound send by GateType.String(); if these
	// three ever diverge, every outbound message to a custom gate is silently
	// unroutable.
	if p.Type() != sharedmodel.TypeCustom.String() {
		t.Fatalf("Type() = %q, GateType = %q", p.Type(), sharedmodel.TypeCustom.String())
	}

	if p.Type() != custommodel.ProviderType {
		t.Fatalf("Type() = %q, ProviderType = %q", p.Type(), custommodel.ProviderType)
	}
}
