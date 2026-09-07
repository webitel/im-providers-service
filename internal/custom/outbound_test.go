package custom

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/google/uuid"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	custommodel "github.com/webitel/im-providers-service/internal/custom/model"
)

type capturedCall struct {
	body      []byte
	signature string
	envelope  envelope
}

// callbackServer stands in for the customer's endpoint.
type callbackServer struct {
	*httptest.Server

	mu     sync.Mutex
	calls  []capturedCall
	status int
	reply  string
}

func newCallbackServer(t *testing.T) *callbackServer {
	t.Helper()

	cs := &callbackServer{status: http.StatusOK, reply: `{"success":true}`}
	cs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)

		var env envelope

		_ = json.Unmarshal(body, &env)

		cs.mu.Lock()
		cs.calls = append(cs.calls, capturedCall{
			body:      body,
			signature: r.Header.Get(signHeader),
			envelope:  env,
		})
		status, reply := cs.status, cs.reply
		cs.mu.Unlock()

		w.WriteHeader(status)
		_, _ = w.Write([]byte(reply))
	}))

	t.Cleanup(cs.Close)

	return cs
}

func (cs *callbackServer) captured() []capturedCall {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	return append([]capturedCall(nil), cs.calls...)
}

func (cs *callbackServer) refuse(reason string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.reply = `{"success":false,"error":"` + reason + `"}`
}

func outboundFixture(t *testing.T, chats map[string]*custommodel.Chat) (*customProvider, *stubStore, *callbackServer) {
	t.Helper()

	server := newCallbackServer(t)

	gate := testGate()
	gate.CallbackURL = server.URL

	store := &stubStore{gate: gate, chatsByRecipient: chats}
	if store.chatsByID == nil {
		store.chatsByID = map[string]*custommodel.Chat{}
	}

	for _, chat := range chats {
		store.chatsByID[chat.ChatID] = chat
	}

	p := newTestProvider(t, store, &recordedMessenger{})
	// The queue is built without starting its workers: these tests assert what
	// lands in it, not how it drains.
	shards := make([]chan retryTask, retryShards)
	for i := range shards {
		shards[i] = make(chan retryTask, retryQueueDepth)
	}

	p.retries = &retryQueue{logger: noopLogger, api: p.api, shards: shards}

	return p, store, server
}

func knownChat() map[string]*custommodel.Chat {
	return map[string]*custommodel.Chat{
		"viber|u1": {GateID: testGateID, ChatID: "c1", ExternalSub: "viber|u1"},
	}
}

func operatorMessage() *sharedmodel.Message {
	return &sharedmodel.Message{
		ID:         uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		GateID:     testGateID,
		DomainID:   7,
		From:       sharedmodel.Peer{Sub: "agent-sub"},
		To:         sharedmodel.Peer{Sub: "viber|u1"},
		Text:       "How can I help?",
		SenderName: "Support",
	}
}

func TestSendText_DeliversSignedMessageToTheKnownChat(t *testing.T) {
	p, _, server := outboundFixture(t, knownChat())

	resp, err := p.SendText(t.Context(), operatorMessage())
	if err != nil {
		t.Fatalf("SendText: %v", err)
	}

	calls := server.captured()
	if len(calls) != 1 {
		t.Fatalf("want 1 callback, got %d", len(calls))
	}

	call := calls[0]
	if call.signature != signature(string(call.body), testSecret) {
		t.Error("outbound payload is not signed with the gate secret")
	}

	msg := call.envelope.Message
	if msg == nil {
		t.Fatal("payload carries no message")
	}

	if msg.ChatID != "c1" {
		t.Errorf("chatId = %q, want c1", msg.ChatID)
	}

	if msg.Text != "How can I help?" {
		t.Errorf("text = %q", msg.Text)
	}

	if msg.Sender == nil || msg.Sender.Name != "Support" {
		t.Errorf("sender = %+v, want the operator's chat name", msg.Sender)
	}

	if msg.Sender.Type != webitelSenderType {
		t.Errorf("sender type = %q, want %q", msg.Sender.Type, webitelSenderType)
	}

	// The id on the wire is the internal message id, which is what makes a
	// menu callback correlate without extra state.
	if msg.ID != "33333333-3333-3333-3333-333333333333" {
		t.Errorf("message id = %q, want the internal message uuid", msg.ID)
	}

	if resp.ID != msg.ID {
		t.Errorf("reported external id = %q, want %q", resp.ID, msg.ID)
	}

	if call.envelope.Broadcast != nil {
		t.Error("a reply into an existing chat must not be a broadcast")
	}
}

func TestSendText_UnknownRecipientBecomesBroadcast(t *testing.T) {
	p, _, server := outboundFixture(t, nil)

	if _, err := p.SendText(t.Context(), operatorMessage()); err != nil {
		t.Fatalf("SendText: %v", err)
	}

	calls := server.captured()
	if len(calls) != 1 {
		t.Fatalf("want 1 callback, got %d", len(calls))
	}

	bc := calls[0].envelope.Broadcast
	if bc == nil {
		t.Fatal("a message to a recipient who never wrote must go out as a broadcast")
	}

	if bc.Text != "How can I help?" {
		t.Errorf("broadcast text = %q", bc.Text)
	}

	if len(bc.Recipients) != 1 {
		t.Fatalf("want 1 recipient, got %d", len(bc.Recipients))
	}

	if bc.Recipients[0].ID != "u1" || bc.Recipients[0].Type != "viber" {
		t.Errorf("recipient = %+v, want id u1 of type viber", bc.Recipients[0])
	}

	if bc.EventID == "" {
		t.Error("broadcast carries no eventId to correlate a failure with")
	}
}

func TestSendDocument_CarriesFileAndCaption(t *testing.T) {
	p, _, server := outboundFixture(t, knownChat())

	msg := operatorMessage()
	msg.Text = "the invoice"
	msg.Documents = []*sharedmodel.Document{{
		URL: "https://storage.example.org/f/1?sig=x", MimeType: "application/pdf", Size: 1024, FileName: "invoice.pdf",
	}}

	if _, err := p.SendDocument(t.Context(), msg); err != nil {
		t.Fatalf("SendDocument: %v", err)
	}

	file := server.captured()[0].envelope.Message.File
	if file == nil {
		t.Fatal("payload carries no file")
	}

	if file.URL != "https://storage.example.org/f/1?sig=x" || file.Name != "invoice.pdf" || file.Mime != "application/pdf" {
		t.Errorf("file = %+v", file)
	}

	if got := server.captured()[0].envelope.Message.Text; got != "the invoice" {
		t.Errorf("caption = %q", got)
	}
}

func TestSendInteractive_MapsMenuRows(t *testing.T) {
	p, _, server := outboundFixture(t, knownChat())

	msg := operatorMessage()
	msg.Text = "Rate the chat"
	msg.Interactive = &sharedmodel.Interactive{
		Body:      "Rate the chat",
		SingleUse: true,
		Placement: sharedmodel.MenuPlacementInline,
		Markup: &sharedmodel.KeyboardMarkup{Rows: []sharedmodel.KeyboardRow{{
			Buttons: []sharedmodel.KeyboardButton{
				{ID: "rate_5", Label: "5", Callback: &sharedmodel.KeyboardButtonCallback{Data: "5"}},
				{ID: "site", Label: "Open", URL: &sharedmodel.KeyboardButtonURL{URL: "https://example.org"}},
				{ID: "geo", Label: "Share", Request: &sharedmodel.KeyboardButtonRequest{Action: "location"}},
			},
		}}},
	}

	if _, err := p.SendInteractive(t.Context(), msg); err != nil {
		t.Fatalf("SendInteractive: %v", err)
	}

	menu := server.captured()[0].envelope.Message.Menu
	if menu == nil {
		t.Fatal("payload carries no menu")
	}

	if !menu.SingleUse || menu.Placement != "inline" {
		t.Errorf("menu = %+v", menu)
	}

	if len(menu.Rows) != 1 || len(menu.Rows[0]) != 3 {
		t.Fatalf("rows = %+v", menu.Rows)
	}

	if got := menu.Rows[0][0]; got.Code != "rate_5" || got.Data != "5" {
		t.Errorf("callback button = %+v", got)
	}

	if got := menu.Rows[0][1]; got.URL != "https://example.org" {
		t.Errorf("url button = %+v", got)
	}

	if got := menu.Rows[0][2]; got.Action != "location" {
		t.Errorf("request button = %+v", got)
	}
}

func TestSendInteractive_RefusesAMenuWithoutButtons(t *testing.T) {
	p, _, _ := outboundFixture(t, knownChat())

	msg := operatorMessage()
	msg.Interactive = &sharedmodel.Interactive{Body: "nothing to press"}

	if _, err := p.SendInteractive(t.Context(), msg); err == nil {
		t.Fatal("an interactive message with no buttons was accepted")
	}
}

// A refusal is the external system's own verdict on the payload; replaying it
// would only be refused again, so it surfaces to the operator now.
func TestSendText_RefusalIsReportedImmediately(t *testing.T) {
	p, _, server := outboundFixture(t, knownChat())
	server.refuse("unknown chat")

	_, err := p.SendText(t.Context(), operatorMessage())
	if !errors.Is(err, custommodel.ErrCallbackRejected) {
		t.Fatalf("want ErrCallbackRejected, got %v", err)
	}
}

// An unreachable endpoint is a transport problem, so the message keeps its
// reference and the retry queue owns the outcome from here.
func TestSendText_UnreachableEndpointIsQueuedNotFailed(t *testing.T) {
	p, _, server := outboundFixture(t, knownChat())
	server.Close()

	resp, err := p.SendText(t.Context(), operatorMessage())
	if err != nil {
		t.Fatalf("an unreachable endpoint must not fail the send inline: %v", err)
	}

	if resp == nil || resp.ID == "" {
		t.Fatal("no external id reported, so the message would have no reference to fail later")
	}

	queued := 0
	for _, shard := range p.retries.shards {
		queued += len(shard)
	}

	if queued != 1 {
		t.Fatalf("want the message queued for retry, queued = %d", queued)
	}
}

func TestSendText_EmptyResponseBodyCountsAsAccepted(t *testing.T) {
	p, _, server := outboundFixture(t, knownChat())

	server.mu.Lock()
	server.reply = ""
	server.mu.Unlock()

	if _, err := p.SendText(t.Context(), operatorMessage()); err != nil {
		t.Fatalf("an empty 200 must count as delivered: %v", err)
	}
}

func TestSendText_QuotedResponseFlagIsAccepted(t *testing.T) {
	p, _, server := outboundFixture(t, knownChat())

	server.mu.Lock()
	server.reply = `{"success":"true"}`
	server.mu.Unlock()

	if _, err := p.SendText(t.Context(), operatorMessage()); err != nil {
		t.Fatalf("the documented quoted success flag was rejected: %v", err)
	}
}

func TestSendText_NonSuccessStatusIsRetryable(t *testing.T) {
	p, _, server := outboundFixture(t, knownChat())

	server.mu.Lock()
	server.status = http.StatusBadGateway
	server.mu.Unlock()

	if _, err := p.SendText(t.Context(), operatorMessage()); err != nil {
		t.Fatalf("a 502 must be queued, not failed inline: %v", err)
	}

	queued := 0
	for _, shard := range p.retries.shards {
		queued += len(shard)
	}

	if queued != 1 {
		t.Fatalf("want the message queued for retry, queued = %d", queued)
	}
}
