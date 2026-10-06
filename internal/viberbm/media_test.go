package viberbm

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	vibbmmodel "github.com/webitel/im-providers-service/internal/viberbm/model"
)

type discardUploader struct{}

func (discardUploader) UploadFile(_ context.Context, _ sharedmodel.UploadRequest, body io.Reader) (sharedmodel.UploadResponse, error) {
	_, _ = io.Copy(io.Discard, body)

	return sharedmodel.UploadResponse{ID: "1"}, nil
}

func TestDownloadInboundMedia_APIKeyOnlyToGateHost(t *testing.T) {
	var gotAuth []string

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = append(gotAuth, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("jpeg"))
	}))
	t.Cleanup(srv.Close)

	cases := []struct {
		name     string
		baseURL  string
		wantAuth string
	}{
		{"gate host gets the key", srv.URL, "App secret-key"},
		{"foreign host gets no key", "https://xyz.api.infobip.com", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotAuth = nil

			p := &viberBMProvider{linkClient: srv.Client(), media: discardUploader{}}
			gate := &vibbmmodel.ViberBMGate{APIKey: "secret-key", BaseURL: tc.baseURL}

			if _, err := p.downloadInboundMedia(context.Background(), gate, srv.URL+"/media/a.jpg", "a.jpg"); err != nil {
				t.Fatalf("download: %v", err)
			}

			if len(gotAuth) != 1 || gotAuth[0] != tc.wantAuth {
				t.Fatalf("Authorization = %q, want one request with %q", gotAuth, tc.wantAuth)
			}
		})
	}
}
