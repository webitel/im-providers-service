package viber

import (
	"testing"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/im-providers-service/internal/provider/format"
)

const zwnj = "‌"

func TestRenderViber(t *testing.T) {
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
			name: "bold",
			text: "hello world",
			entities: []sharedmodel.Entity{
				{Type: format.TypeBold, Offset: 0, Length: 5},
			},
			want: "*hello* world",
		},
		{
			name: "italic",
			text: "hello world",
			entities: []sharedmodel.Entity{
				{Type: format.TypeItalic, Offset: 6, Length: 5},
			},
			want: "hello _world_",
		},
		{
			name: "strikethrough",
			text: "hello world",
			entities: []sharedmodel.Entity{
				{Type: format.TypeStrikethrough, Offset: 0, Length: 5},
			},
			want: "~hello~ world",
		},
		{
			name: "monospace",
			text: "hello world",
			entities: []sharedmodel.Entity{
				{Type: format.TypeCode, Offset: 0, Length: 5},
			},
			want: "```hello``` world",
		},
		{
			name: "link with no dedicated markup keeps its URL visible as a parenthetical",
			text: "click here",
			entities: []sharedmodel.Entity{
				{Type: format.TypeLink, Offset: 6, Length: 4, Value: "https://example.com"},
			},
			want: "click here (https://example.com)",
		},
		{
			name:     "literal marker at a word boundary is escaped",
			text:     "hello * world",
			entities: nil,
			want:     "hello *‌ world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderViber(tt.text, tt.entities)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseViberMarkdown(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "plain text passes through unchanged",
			text: "hello world",
			want: "hello world",
		},
		{
			name: "Viber's own markdown markers pass through unchanged",
			text: "*hello* _world_ ~foo~ ```bar```",
			want: "*hello* _world_ ~foo~ ```bar```",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseViberMarkdown(tt.text)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
