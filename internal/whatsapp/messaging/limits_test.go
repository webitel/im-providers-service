package messaging

import (
	"testing"

	"github.com/webitel/im-providers-service/internal/core/model"
)

func TestCheckImageSize(t *testing.T) {
	cases := []struct {
		name string
		size int64
		want string
	}{
		{"unknown size", 0, ""},
		{"at limit", maxImageBytes, ""},
		{"over limit", maxImageBytes + 1, model.SendErrFileTooLarge},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := model.ClassifySendError(checkImageSize([]*model.Image{{Size: tc.size}})); got != tc.want {
				t.Errorf("checkImageSize(%d) = %q, want %q", tc.size, got, tc.want)
			}
		})
	}
}

func TestCheckDocumentSize(t *testing.T) {
	cases := []struct {
		name string
		size int64
		want string
	}{
		{"unknown size", 0, ""},
		{"image limit does not apply", maxImageBytes + 1, ""},
		{"at limit", maxDocumentBytes, ""},
		{"over limit", maxDocumentBytes + 1, model.SendErrFileTooLarge},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := model.ClassifySendError(checkDocumentSize([]*model.Document{{Size: tc.size}})); got != tc.want {
				t.Errorf("checkDocumentSize(%d) = %q, want %q", tc.size, got, tc.want)
			}
		})
	}

	if err := checkDocumentSize(nil); err != nil {
		t.Errorf("no documents must pass, got %v", err)
	}
}
