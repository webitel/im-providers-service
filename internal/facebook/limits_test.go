package facebook

import (
	"testing"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

func TestCheckFirstSize(t *testing.T) {
	cases := []struct {
		name string
		docs []*sharedmodel.Document
		want string
	}{
		{"no attachments", nil, ""},
		{"unknown size", []*sharedmodel.Document{{Size: 0}}, ""},
		{"at limit", []*sharedmodel.Document{{Size: maxAttachmentBytes}}, ""},
		{"over limit", []*sharedmodel.Document{{Size: maxAttachmentBytes + 1}}, sharedmodel.SendErrFileTooLarge},
		{"only the sent attachment counts", []*sharedmodel.Document{{Size: 1}, {Size: maxAttachmentBytes + 1}}, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sharedmodel.ClassifySendError(checkFirstSize("messenger file", tc.docs)); got != tc.want {
				t.Errorf("checkFirstSize = %q, want %q", got, tc.want)
			}
		})
	}

	images := []*sharedmodel.Image{{Size: maxAttachmentBytes + 1}}
	if got := sharedmodel.ClassifySendError(checkFirstSize("messenger image", images)); got != sharedmodel.SendErrFileTooLarge {
		t.Errorf("oversized image classified as %q", got)
	}
}
