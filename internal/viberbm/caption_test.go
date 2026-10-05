package viberbm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	vibbmmodel "github.com/webitel/im-providers-service/internal/viberbm/model"
)

func TestSendDocument_FileCaptionFollowUp(t *testing.T) {
	const okBody = `{"bulkId":"b","messages":[{"messageId":"file-msg","status":{"groupId":1,"groupName":"PENDING","name":"PENDING_ENROUTE","description":""}}]}`

	cases := []struct {
		name        string
		caption     string
		textStatus  int
		wantContent []string
	}{
		{"caption sent as TEXT after FILE", "see attached", http.StatusOK, []string{contentTypeFile, contentTypeText}},
		{"blank caption sends only FILE", "  ", http.StatusOK, []string{contentTypeFile}},
		{"caption failure does not fail the send", "see attached", http.StatusBadRequest, []string{contentTypeFile, contentTypeText}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				mu     sync.Mutex
				bodies []envelope
			)

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				raw, _ := io.ReadAll(r.Body)

				var env envelope

				_ = json.Unmarshal(raw, &env)

				mu.Lock()

				bodies = append(bodies, env)
				n := len(bodies)

				mu.Unlock()

				w.Header().Set("Content-Type", "application/json")

				if n > 1 && tc.textStatus != http.StatusOK {
					w.WriteHeader(tc.textStatus)
					_, _ = io.WriteString(w, `{"errorCode":"E400","description":"bad"}`)

					return
				}

				_, _ = io.WriteString(w, okBody)
			}))
			t.Cleanup(srv.Close)

			p := &viberBMProvider{
				api:    apiClientForServer(srv),
				logger: discardLogger(),
				repo:   &signatureRepo{gate: &vibbmmodel.ViberBMGate{ID: "g1", BaseURL: srv.URL, APIKey: "key", SenderName: "Sender"}},
			}

			resp, err := p.SendDocument(context.Background(), &sharedmodel.Message{
				GateID:    "g1",
				To:        sharedmodel.Peer{Sub: "38599000001"},
				Text:      tc.caption,
				Documents: []*sharedmodel.Document{{URL: "https://example.com/r.pdf", FileName: "report.pdf", MimeType: "application/pdf"}},
			})
			if err != nil {
				t.Fatalf("SendDocument: %v", err)
			}

			if resp.ID != "file-msg" {
				t.Errorf("response id = %q, want the FILE message id", resp.ID)
			}

			if len(bodies) != len(tc.wantContent) {
				t.Fatalf("requests = %d, want %d", len(bodies), len(tc.wantContent))
			}

			for i, want := range tc.wantContent {
				content, _ := bodies[i].Messages[0].Content.(map[string]any)
				if content["type"] != want {
					t.Errorf("request %d content.type = %v, want %s", i, content["type"], want)
				}
			}

			if len(bodies) == 2 {
				content, _ := bodies[1].Messages[0].Content.(map[string]any)
				if content["text"] != tc.caption {
					t.Errorf("caption text = %v, want %q", content["text"], tc.caption)
				}

				if bodies[1].Messages[0].MessageID != "" {
					t.Errorf("caption messageId = %q, want empty", bodies[1].Messages[0].MessageID)
				}
			}
		})
	}
}
