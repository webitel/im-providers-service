package webhook

import "testing"

func TestParseWhatsAppMarkdown(t *testing.T) {
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
			name: "WhatsApp's own markdown markers pass through unchanged",
			text: "*hello* _world_ ~foo~ ```bar```",
			want: "*hello* _world_ ~foo~ ```bar```",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseWhatsAppMarkdown(tt.text)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
