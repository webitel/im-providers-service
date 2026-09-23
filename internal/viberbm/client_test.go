package viberbm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// capturedRequest holds what the test server received so assertions can check
// both the wire JSON and the HTTP headers.
type capturedRequest struct {
	header http.Header
	body   []byte
}

// apiClientForServer returns an apiClient whose HTTP transport hits srv.
func apiClientForServer(srv *httptest.Server) *apiClient {
	return &apiClient{
		http:   srv.Client(),
		logger: discardLogger(),
	}
}

// serveJSON starts a one-shot test server that returns statusCode with body.
// The incoming request is captured into *capturedRequest.
func serveJSON(t *testing.T, statusCode int, body string) (*httptest.Server, *capturedRequest) {
	t.Helper()

	cap := &capturedRequest{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.header = r.Header.Clone()
		cap.body, _ = io.ReadAll(r.Body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = io.WriteString(w, body)
	}))

	t.Cleanup(srv.Close)

	return srv, cap
}

// ---------------------------------------------------------------------------
// Authorization header
// ---------------------------------------------------------------------------

func TestAPIClient_AuthorizationHeader(t *testing.T) {
	respBody := `{"bulkId":"b1","messages":[{"messageId":"m1","status":{"groupId":1,"groupName":"PENDING","name":"PENDING_ENROUTE","description":""}}]}`
	srv, cap := serveJSON(t, 200, respBody)

	c := apiClientForServer(srv)

	_, _ = c.SendText(context.Background(), srv.URL, "my-api-key", "MySender", "38599111", "hello", "")

	got := cap.header.Get("Authorization")
	if got != "App my-api-key" {
		t.Errorf("Authorization = %q, want %q", got, "App my-api-key")
	}
}

// ---------------------------------------------------------------------------
// SendText — golden JSON envelope
// ---------------------------------------------------------------------------

func TestAPIClient_SendText_Envelope(t *testing.T) {
	respBody := `{"bulkId":"bulk-123","messages":[{"messageId":"msg-abc","status":{"groupId":1,"groupName":"PENDING","name":"PENDING_ENROUTE","description":""}}]}`
	srv, cap := serveJSON(t, 200, respBody)

	c := apiClientForServer(srv)

	res, err := c.SendText(context.Background(), srv.URL, "key", "BotName", "38599000001", "hello world", "idem-1")
	if err != nil {
		t.Fatalf("SendText: %v", err)
	}

	// Decode what the server received.
	var env envelope
	if err := json.Unmarshal(cap.body, &env); err != nil {
		t.Fatalf("parse captured body: %v", err)
	}

	if len(env.Messages) != 1 {
		t.Fatalf("expected 1 message in envelope, got %d", len(env.Messages))
	}

	msg := env.Messages[0]

	if msg.Sender != "BotName" {
		t.Errorf("sender = %q, want BotName", msg.Sender)
	}

	if len(msg.Destinations) != 1 || msg.Destinations[0].To != "38599000001" {
		t.Errorf("destinations = %+v, want [{To:38599000001}]", msg.Destinations)
	}

	if msg.MessageID != "idem-1" {
		t.Errorf("messageId = %q, want idem-1", msg.MessageID)
	}

	tc, ok := msg.Content.(map[string]any)
	if !ok {
		t.Fatalf("content not a map: %T", msg.Content)
	}

	if got := tc["type"]; got != contentTypeText {
		t.Errorf("content.type = %v, want %s", got, contentTypeText)
	}

	if got := tc["text"]; got != "hello world" {
		t.Errorf("content.text = %v, want hello world", got)
	}

	// Response decode.
	if res.BulkID != "bulk-123" {
		t.Errorf("BulkID = %q, want bulk-123", res.BulkID)
	}

	if res.MessageID != "msg-abc" {
		t.Errorf("MessageID = %q, want msg-abc", res.MessageID)
	}
}

// ---------------------------------------------------------------------------
// SendImage — golden envelope
// ---------------------------------------------------------------------------

func TestAPIClient_SendImage_Envelope(t *testing.T) {
	respBody := `{"bulkId":"b2","messages":[{"messageId":"m2","status":{"groupId":1,"groupName":"PENDING","name":"PENDING_ENROUTE","description":""}}]}`
	srv, cap := serveJSON(t, 200, respBody)

	c := apiClientForServer(srv)

	_, err := c.SendImage(context.Background(), srv.URL, "key", "Sender", "38599", "https://example.com/img.jpg", "caption text", "idem-2")
	if err != nil {
		t.Fatalf("SendImage: %v", err)
	}

	var env envelope
	if err := json.Unmarshal(cap.body, &env); err != nil {
		t.Fatalf("parse body: %v", err)
	}

	tc, _ := env.Messages[0].Content.(map[string]any)

	if got := tc["type"]; got != contentTypeImage {
		t.Errorf("content.type = %v, want %s", got, contentTypeImage)
	}

	if got := tc["mediaUrl"]; got != "https://example.com/img.jpg" {
		t.Errorf("mediaUrl = %v, want the image URL", got)
	}

	if got := tc["text"]; got != "caption text" {
		t.Errorf("text (caption) = %v, want caption text", got)
	}
}

// ---------------------------------------------------------------------------
// SendFile — golden envelope
// ---------------------------------------------------------------------------

func TestAPIClient_SendFile_Envelope(t *testing.T) {
	respBody := `{"bulkId":"b3","messages":[{"messageId":"m3","status":{"groupId":1,"groupName":"PENDING","name":"PENDING_ENROUTE","description":""}}]}`
	srv, cap := serveJSON(t, 200, respBody)

	c := apiClientForServer(srv)

	_, err := c.SendFile(context.Background(), srv.URL, "key", "Sender", "38599", "https://example.com/doc.pdf", "report.pdf", "idem-3")
	if err != nil {
		t.Fatalf("SendFile: %v", err)
	}

	var env envelope
	if err := json.Unmarshal(cap.body, &env); err != nil {
		t.Fatalf("parse body: %v", err)
	}

	tc, _ := env.Messages[0].Content.(map[string]any)

	if got := tc["type"]; got != contentTypeFile {
		t.Errorf("content.type = %v, want %s", got, contentTypeFile)
	}

	if got := tc["mediaUrl"]; got != "https://example.com/doc.pdf" {
		t.Errorf("mediaUrl = %v", got)
	}

	if got := tc["fileName"]; got != "report.pdf" {
		t.Errorf("fileName = %v", got)
	}
}

// ---------------------------------------------------------------------------
// SendTemplate — golden envelope
// ---------------------------------------------------------------------------

func TestAPIClient_SendTemplate_Envelope(t *testing.T) {
	respBody := `{"bulkId":"b4","messages":[{"messageId":"m4","status":{"groupId":1,"groupName":"PENDING","name":"PENDING_ENROUTE","description":""}}]}`
	srv, cap := serveJSON(t, 200, respBody)

	c := apiClientForServer(srv)
	params := map[string]string{"name": "Alice"}

	_, err := c.SendTemplate(context.Background(), srv.URL, "key", "Sender", "38599", "tpl-42", "en", params, "idem-4")
	if err != nil {
		t.Fatalf("SendTemplate: %v", err)
	}

	var env envelope
	if err := json.Unmarshal(cap.body, &env); err != nil {
		t.Fatalf("parse body: %v", err)
	}

	tc, _ := env.Messages[0].Content.(map[string]any)

	if got := tc["type"]; got != contentTypeTemplate {
		t.Errorf("content.type = %v, want %s", got, contentTypeTemplate)
	}

	if got := tc["templateId"]; got != "tpl-42" {
		t.Errorf("templateId = %v", got)
	}

	if got := tc["language"]; got != "en" {
		t.Errorf("language = %v", got)
	}
}

// ---------------------------------------------------------------------------
// messageId is echoed back (idempotency)
// ---------------------------------------------------------------------------

func TestAPIClient_SendText_MessageIDEchoed(t *testing.T) {
	const idempotencyKey = "unique-idempotency-key-xyz"

	respBody := `{"bulkId":"bx","messages":[{"messageId":"` + idempotencyKey + `","status":{"groupId":1,"groupName":"PENDING","name":"PENDING_ENROUTE","description":""}}]}`
	srv, cap := serveJSON(t, 200, respBody)

	c := apiClientForServer(srv)

	res, err := c.SendText(context.Background(), srv.URL, "k", "S", "38599", "hi", idempotencyKey)
	if err != nil {
		t.Fatalf("SendText: %v", err)
	}

	// Verify the key was sent in the wire envelope.
	var env envelope

	_ = json.Unmarshal(cap.body, &env)

	if env.Messages[0].MessageID != idempotencyKey {
		t.Errorf("sent messageId = %q, want %q", env.Messages[0].MessageID, idempotencyKey)
	}

	// Verify the response carries it back.
	if res.MessageID != idempotencyKey {
		t.Errorf("response MessageID = %q, want %q", res.MessageID, idempotencyKey)
	}
}

// ---------------------------------------------------------------------------
// non-2xx with InfoBip error body
// ---------------------------------------------------------------------------

func TestAPIClient_Send_Non2xx_InfoBipError(t *testing.T) {
	errBody := `{"requestError":{"serviceException":{"messageId":"BAD_REQUEST","text":"Invalid sender"}}}`
	srv, _ := serveJSON(t, 400, errBody)

	c := apiClientForServer(srv)

	_, err := c.SendText(context.Background(), srv.URL, "k", "S", "38599", "hi", "")
	if err == nil {
		t.Fatal("expected error for 400 response, got nil")
	}

	if !strings.Contains(err.Error(), "Invalid sender") {
		t.Errorf("error should contain InfoBip text; got: %v", err)
	}

	if !strings.Contains(err.Error(), "400") {
		t.Errorf("error should contain HTTP status code; got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// non-2xx without InfoBip body — generic wrap
// ---------------------------------------------------------------------------

func TestAPIClient_Send_Non2xx_GenericError(t *testing.T) {
	srv, _ := serveJSON(t, 500, `{"bulkId":"","messages":[]}`)

	c := apiClientForServer(srv)

	_, err := c.SendText(context.Background(), srv.URL, "k", "S", "38599", "hi", "")
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}

	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should contain 500; got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 2xx but REJECTED status in messages[0]
// ---------------------------------------------------------------------------

func TestAPIClient_Send_2xx_ButRejected(t *testing.T) {
	respBody := `{"bulkId":"bx","messages":[{"messageId":"mx","status":{"groupId":5,"groupName":"REJECTED","name":"REJECTED_OPERATOR","description":"number barred"}}]}`
	srv, _ := serveJSON(t, 200, respBody)

	c := apiClientForServer(srv)

	_, err := c.SendText(context.Background(), srv.URL, "k", "S", "38599", "hi", "")
	if err == nil {
		t.Fatal("expected error for REJECTED message status, got nil")
	}

	if !strings.Contains(err.Error(), "rejected") {
		t.Errorf("error should mention rejection; got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 2xx but empty messages array
// ---------------------------------------------------------------------------

func TestAPIClient_Send_2xx_EmptyMessages(t *testing.T) {
	srv, _ := serveJSON(t, 200, `{"bulkId":"bx","messages":[]}`)

	c := apiClientForServer(srv)

	_, err := c.SendText(context.Background(), srv.URL, "k", "S", "38599", "hi", "")
	if err == nil {
		t.Fatal("expected error for empty messages array, got nil")
	}

	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should mention 'empty'; got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// URL construction — base URL trailing slash stripped
// ---------------------------------------------------------------------------

func TestAPIClient_SendText_URLConstruction(t *testing.T) {
	cases := []struct {
		name    string
		baseURL string
	}{
		{"no trailing slash", "BASE_URL"},
		{"trailing slash stripped", "BASE_URL/"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var capturedPath string

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedPath = r.URL.Path

				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"bulkId":"b","messages":[{"messageId":"m","status":{"groupId":1,"groupName":"PENDING","name":"P","description":""}}]}`)
			}))
			t.Cleanup(srv.Close)

			c := apiClientForServer(srv)
			// Replace BASE_URL with the actual server URL.
			base := strings.Replace(tc.baseURL, "BASE_URL", srv.URL, 1)
			_, _ = c.SendText(context.Background(), base, "k", "S", "38599", "hi", "")

			if capturedPath != sendMessagesPath {
				t.Errorf("path = %q, want %q", capturedPath, sendMessagesPath)
			}
		})
	}
}
