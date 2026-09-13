package facebook

import (
	"testing"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/im-providers-service/internal/provider/format"
)

func TestRenderFacebook(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		entities []sharedmodel.Entity
		want     string
	}{
		{
			name:     "no entities",
			text:     "hello world",
			entities: nil,
			want:     "hello world",
		},
		{
			name: "bold/italic markers degrade, undocumented by Send API",
			text: "hello world",
			entities: []sharedmodel.Entity{
				{Type: format.TypeBold, Offset: 0, Length: 5},
				{Type: format.TypeItalic, Offset: 6, Length: 5},
			},
			want: "hello world",
		},
		{
			name: "link keeps its URL visible as a parenthetical",
			text: "click here",
			entities: []sharedmodel.Entity{
				{Type: format.TypeLink, Offset: 6, Length: 4, Value: "https://example.com"},
			},
			want: "click here (https://example.com)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderFacebook(tt.text, tt.entities)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

const zwnj = "‌"

func TestParseFacebookMarkdown(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "plain text is unaffected",
			text: "hello world",
			want: "hello world",
		},
		{
			name: "a literal asterisk the user typed is escaped, not rendered as bold",
			text: "hello * world",
			want: "hello *" + zwnj + " world",
		},
		{
			name: "a marker between two letters is left untouched",
			text: "a_b_c",
			want: "a_b_c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseFacebookMarkdown(tt.text)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
