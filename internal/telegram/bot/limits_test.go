package bot

import (
	"testing"

	coremodel "github.com/webitel/im-providers-service/internal/core/model"
)

func TestCheckImageSizes(t *testing.T) {
	cases := []struct {
		name string
		size int64
		want string
	}{
		{"unknown size", 0, ""},
		{"at limit", maxPhotoURLBytes, ""},
		{"over limit", maxPhotoURLBytes + 1, coremodel.SendErrFileTooLarge},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			images := []*coremodel.Image{{Size: 1}, nil, {Size: tc.size}}
			if got := coremodel.ClassifySendError(checkImageSizes(images)); got != tc.want {
				t.Errorf("checkImageSizes(%d) = %q, want %q", tc.size, got, tc.want)
			}
		})
	}
}

func TestCheckDocumentSizes(t *testing.T) {
	cases := []struct {
		name string
		size int64
		want string
	}{
		{"unknown size", 0, ""},
		{"at limit", maxDocumentURLBytes, ""},
		{"photo limit does not apply", maxPhotoURLBytes + 1, ""},
		{"over limit", maxDocumentURLBytes + 1, coremodel.SendErrFileTooLarge},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			docs := []*coremodel.Document{{Size: 1}, nil, {Size: tc.size}}
			if got := coremodel.ClassifySendError(checkDocumentSizes(docs)); got != tc.want {
				t.Errorf("checkDocumentSizes(%d) = %q, want %q", tc.size, got, tc.want)
			}
		})
	}
}
