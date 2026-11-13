package bot

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	coremodel "github.com/webitel/im-providers-service/internal/core/model"
	tgclient "github.com/webitel/im-providers-service/internal/telegram/bot/client"
	"github.com/webitel/im-providers-service/internal/telegram/bot/model"
)

const testBotToken = "123456:test-token-telegram-inbound"

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type fakeMediaManager struct {
	req  coremodel.UploadRequest
	body string
}

func (f *fakeMediaManager) UploadFile(_ context.Context, req coremodel.UploadRequest, body io.Reader) (coremodel.UploadResponse, error) {
	f.req = req
	data, _ := io.ReadAll(body)
	f.body = string(data)

	return coremodel.UploadResponse{ID: "131353", Size: int64(len(data))}, nil
}

func telegramFiles(filePath string) *tgclient.Client {
	return tgclient.NewClientWithHTTP(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := "file-bytes"
		if strings.HasSuffix(r.URL.Path, "/getFile") {
			body = `{"ok":true,"result":{"file_id":"f1","file_unique_id":"u1","file_size":10,"file_path":"` + filePath + `"}}`
		}

		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
	})})
}

func TestSyncFile_UploadsBodyAndHidesToken(t *testing.T) {
	media := &fakeMediaManager{}
	p := &Provider{tgMessageClient: telegramFiles("documents/file_7.pdf"), mediaManager: media}
	gate := &model.Gate{Token: testBotToken, DC: 1}

	got, err := p.syncFile(context.Background(), gate, "f1", "", "", nil)
	if err != nil {
		t.Fatalf("syncFile: %v", err)
	}

	if got.id != "131353" {
		t.Errorf("id = %q, want storage id", got.id)
	}

	if got.name != "file_7.pdf" || got.mimeType != "application/pdf" {
		t.Errorf("name/mime = %q/%q", got.name, got.mimeType)
	}

	if got.size != 10 {
		t.Errorf("size = %d, want 10", got.size)
	}

	if media.body != "file-bytes" {
		t.Errorf("uploaded body = %q", media.body)
	}

	if strings.Contains(media.req.URL, "test-token") || media.req.URL != "" {
		t.Errorf("upload request carries a url: %q", media.req.URL)
	}
}

func TestSyncFile_KeepsTelegramMetadata(t *testing.T) {
	media := &fakeMediaManager{}
	p := &Provider{tgMessageClient: telegramFiles("documents/file_7"), mediaManager: media}
	size := int64(2048)

	got, err := p.syncFile(context.Background(), &model.Gate{Token: testBotToken, DC: 1}, "f1", "report.docx", "application/msword", &size)
	if err != nil {
		t.Fatalf("syncFile: %v", err)
	}

	if got.name != "report.docx" || got.mimeType != "application/msword" || got.size != 2048 {
		t.Errorf("got %+v", got)
	}
}

func TestLargestPhotoSize(t *testing.T) {
	sizes := []model.PhotoSize{
		{FileID: "s", Width: 90, Height: 60},
		{FileID: "l", Width: 1280, Height: 853},
		{FileID: "m", Width: 320, Height: 213},
	}

	if got := largestPhotoSize(sizes); got.FileID != "l" {
		t.Errorf("largest = %q, want l", got.FileID)
	}
}
