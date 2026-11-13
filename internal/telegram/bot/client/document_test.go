package client

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
)

func fileTransport(download func(*http.Request) (*http.Response, error)) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if strings.HasSuffix(r.URL.Path, "/"+getFileMethod) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"ok":true,"result":{"file_id":"f1","file_unique_id":"u1","file_size":5,"file_path":"documents/report.pdf"}}`)),
				Header:     http.Header{},
			}, nil
		}

		return download(r)
	})}
}

func TestDownloadFile_ReturnsBody(t *testing.T) {
	var gotPath string

	c := &Client{httpClient: fileTransport(func(r *http.Request) (*http.Response, error) {
		gotPath = r.URL.Path

		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("hello")), Header: http.Header{}}, nil
	})}

	body, file, err := c.DownloadFile(context.Background(), testToken, "f1")
	if err != nil {
		t.Fatalf("DownloadFile: %v", err)
	}
	defer body.Close()

	data, _ := io.ReadAll(body)
	if string(data) != "hello" {
		t.Errorf("body = %q, want hello", data)
	}

	if file.FilePath == nil || *file.FilePath != "documents/report.pdf" {
		t.Errorf("file path = %v", file.FilePath)
	}

	if gotPath != "/file/bot"+testToken+"/documents/report.pdf" {
		t.Errorf("download path = %q", gotPath)
	}
}

func TestDownloadFile_TransportErrorRedactsToken(t *testing.T) {
	c := &Client{httpClient: fileTransport(func(*http.Request) (*http.Response, error) {
		return nil, &net.OpError{Op: "dial", Net: "tcp", Err: timeoutErr{}}
	})}

	_, _, err := c.DownloadFile(context.Background(), testToken, "f1")
	if err == nil {
		t.Fatal("expected a transport error")
	}

	if strings.Contains(err.Error(), "test-token-telegram") {
		t.Fatalf("token leaked: %q", err.Error())
	}
}

func TestDownloadFile_NonOKStatus(t *testing.T) {
	closed := false
	c := &Client{httpClient: fileTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found", Body: closeTracker{&closed}, Header: http.Header{}}, nil
	})}

	_, _, err := c.DownloadFile(context.Background(), testToken, "f1")
	if err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("expected a 404 error, got %v", err)
	}

	if strings.Contains(err.Error(), "test-token-telegram") {
		t.Fatalf("token leaked: %q", err.Error())
	}

	if !closed {
		t.Error("response body not closed")
	}
}

type closeTracker struct{ closed *bool }

func (closeTracker) Read([]byte) (int, error) { return 0, io.EOF }
func (c closeTracker) Close() error {
	*c.closed = true

	return nil
}
