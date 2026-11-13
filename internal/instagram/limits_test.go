package instagram

import (
	"testing"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

func TestCheckImageSize(t *testing.T) {
	cases := []struct {
		name   string
		images []*sharedmodel.Image
		want   string
	}{
		{"no images", nil, ""},
		{"nil image", []*sharedmodel.Image{nil}, ""},
		{"unknown size", []*sharedmodel.Image{{Size: 0}}, ""},
		{"at limit", []*sharedmodel.Image{{Size: maxImageBytes}}, ""},
		{"over limit", []*sharedmodel.Image{{Size: maxImageBytes + 1}}, sharedmodel.SendErrFileTooLarge},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sharedmodel.ClassifySendError(checkImageSize(tc.images)); got != tc.want {
				t.Errorf("checkImageSize = %q, want %q", got, tc.want)
			}
		})
	}
}
