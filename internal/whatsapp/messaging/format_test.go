package messaging

import (
	"testing"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/im-providers-service/internal/provider/format"
)

func TestRenderWhatsApp(t *testing.T) {
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
			name: "link with valid https",
			text: "click here",
			entities: []sharedmodel.Entity{
				{Type: format.TypeLink, Offset: 6, Length: 4, Value: "https://example.com"},
			},
			want: "click here (https://example.com)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderWhatsApp(tt.text, tt.entities)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
