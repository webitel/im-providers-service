package bot

import (
	"testing"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/im-providers-service/internal/provider/format"
	"github.com/webitel/im-providers-service/internal/telegram/bot/model"
)

func strPtr(s string) *string { return &s }

const zwnj = "‌"

func TestRenderTelegramHTML(t *testing.T) {
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
			want: "<b>hello</b> world",
		},
		{
			name: "italic",
			text: "hello world",
			entities: []sharedmodel.Entity{
				{Type: format.TypeItalic, Offset: 6, Length: 5},
			},
			want: "hello <i>world</i>",
		},
		{
			name: "strikethrough",
			text: "hello world",
			entities: []sharedmodel.Entity{
				{Type: format.TypeStrikethrough, Offset: 0, Length: 5},
			},
			want: "<s>hello</s> world",
		},
		{
			name: "code",
			text: "hello world",
			entities: []sharedmodel.Entity{
				{Type: format.TypeCode, Offset: 0, Length: 5},
			},
			want: "<code>hello</code> world",
		},
		{
			name: "pre",
			text: "hello world",
			entities: []sharedmodel.Entity{
				{Type: format.TypePre, Offset: 0, Length: 5},
			},
			want: "<pre>hello</pre> world",
		},
		{
			name: "link with valid https",
			text: "click here",
			entities: []sharedmodel.Entity{
				{Type: format.TypeLink, Offset: 6, Length: 4, Value: "https://example.com"},
			},
			want: `click <a href="https://example.com">here</a>`,
		},
		{
			name: "link with valid http",
			text: "click here",
			entities: []sharedmodel.Entity{
				{Type: format.TypeLink, Offset: 6, Length: 4, Value: "http://example.com"},
			},
			want: `click <a href="http://example.com">here</a>`,
		},
		{
			name: "link with mailto",
			text: "email me",
			entities: []sharedmodel.Entity{
				{Type: format.TypeLink, Offset: 0, Length: 5, Value: "mailto:test@example.com"},
			},
			want: `<a href="mailto:test@example.com">email</a> me`,
		},
		{
			name: "link with invalid scheme",
			text: "click here",
			entities: []sharedmodel.Entity{
				{Type: format.TypeLink, Offset: 6, Length: 4, Value: "javascript:alert(1)"},
			},
			want: "click here",
		},
		{
			name: "link with bare scheme and no host is rejected",
			text: "click here",
			entities: []sharedmodel.Entity{
				{Type: format.TypeLink, Offset: 6, Length: 4, Value: "https:"},
			},
			want: "click here",
		},
		{
			name: "link value containing angle brackets does not break out of the attribute",
			text: "link",
			entities: []sharedmodel.Entity{
				{Type: format.TypeLink, Offset: 0, Length: 4, Value: `https://ex.com/?q=<script>`},
			},
			want: `<a href="https://ex.com/?q=&lt;script&gt;">link</a>`,
		},
		{
			name: "unknown entity type",
			text: "hello world",
			entities: []sharedmodel.Entity{
				{Type: "SPOILER", Offset: 0, Length: 5},
			},
			want: "hello world",
		},
		{
			name: "out of bounds entity",
			text: "hello",
			entities: []sharedmodel.Entity{
				{Type: format.TypeBold, Offset: 0, Length: 10},
			},
			want: "hello",
		},
		{
			name:     "escape ampersand",
			text:     "hello & world",
			entities: nil,
			want:     "hello &amp; world",
		},
		{
			name:     "escape less than",
			text:     "hello < world",
			entities: nil,
			want:     "hello &lt; world",
		},
		{
			name:     "escape greater than",
			text:     "hello > world",
			entities: nil,
			want:     "hello &gt; world",
		},
		{
			name: "escape inside code span",
			text: "a < b",
			entities: []sharedmodel.Entity{
				{Type: format.TypeCode, Offset: 0, Length: 5},
			},
			want: "<code>a &lt; b</code>",
		},
		{
			name: "escape in link href",
			text: "link",
			entities: []sharedmodel.Entity{
				{Type: format.TypeLink, Offset: 0, Length: 4, Value: `https://example.com/path?a=1&b=2`},
			},
			want: `<a href="https://example.com/path?a=1&amp;b=2">link</a>`,
		},
		{
			name: "escape quote in link href",
			text: "link",
			entities: []sharedmodel.Entity{
				{Type: format.TypeLink, Offset: 0, Length: 4, Value: `https://example.com/path?msg="hello"`},
			},
			want: `<a href="https://example.com/path?msg=&quot;hello&quot;">link</a>`,
		},
		{
			name: "multiple entities",
			text: "hello world foo",
			entities: []sharedmodel.Entity{
				{Type: format.TypeBold, Offset: 0, Length: 5},
				{Type: format.TypeItalic, Offset: 6, Length: 5},
				{Type: format.TypeStrikethrough, Offset: 12, Length: 3},
			},
			want: "<b>hello</b> <i>world</i> <s>foo</s>",
		},
		{
			name: "bold nested inside code is dropped (Telegram forbids nested tags in <code>)",
			text: "code",
			entities: []sharedmodel.Entity{
				{Type: format.TypeCode, Offset: 0, Length: 4},
				{Type: format.TypeBold, Offset: 0, Length: 4},
			},
			want: "<code>code</code>",
		},
		{
			name: "link nested inside another link is dropped",
			text: "0123456789",
			entities: []sharedmodel.Entity{
				{Type: format.TypeLink, Offset: 0, Length: 10, Value: "https://outer.example"},
				{Type: format.TypeLink, Offset: 2, Length: 4, Value: "https://inner.example"},
			},
			want: `<a href="https://outer.example">0123456789</a>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderTelegramHTML(tt.text, tt.entities)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEscapeHTMLText(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello world", "hello world"},
		{"a & b", "a &amp; b"},
		{"a < b", "a &lt; b"},
		{"a > b", "a &gt; b"},
		{"a &<> b", "a &amp;&lt;&gt; b"},
	}

	for _, tt := range tests {
		got := escapeHTMLText(tt.input)
		if got != tt.want {
			t.Errorf("escapeHTMLText(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEscapeHTMLAttr(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://example.com", "https://example.com"},
		{"a&b", "a&amp;b"},
		{`a"b`, "a&quot;b"},
		{`a&b"c`, "a&amp;b&quot;c"},
		{"a<b>c", "a&lt;b&gt;c"},
	}

	for _, tt := range tests {
		got := escapeHTMLAttr(tt.input)
		if got != tt.want {
			t.Errorf("escapeHTMLAttr(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseTelegramMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		entities []model.MessageEntity
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
			entities: []model.MessageEntity{
				{Type: "bold", Offset: 0, Length: 5},
			},
			want: "*hello* world",
		},
		{
			name: "italic",
			text: "hello world",
			entities: []model.MessageEntity{
				{Type: "italic", Offset: 6, Length: 5},
			},
			want: "hello _world_",
		},
		{
			name: "strikethrough",
			text: "hello world",
			entities: []model.MessageEntity{
				{Type: "strikethrough", Offset: 0, Length: 5},
			},
			want: "~hello~ world",
		},
		{
			name: "code",
			text: "hello world",
			entities: []model.MessageEntity{
				{Type: "code", Offset: 0, Length: 5},
			},
			want: "```hello``` world",
		},
		{
			name: "pre",
			text: "hello world",
			entities: []model.MessageEntity{
				{Type: "pre", Offset: 0, Length: 5},
			},
			want: "```hello``` world",
		},
		{
			name: "text_link becomes a LINK entity",
			text: "click here",
			entities: []model.MessageEntity{
				{Type: "text_link", Offset: 6, Length: 4, URL: strPtr("https://example.com")},
			},
			want: "click here (https://example.com)",
		},
		{
			name: "text_link without a URL is dropped",
			text: "click here",
			entities: []model.MessageEntity{
				{Type: "text_link", Offset: 6, Length: 4},
			},
			want: "click here",
		},
		{
			name: "unsupported entity type degrades to plain text",
			text: "hello world",
			entities: []model.MessageEntity{
				{Type: "spoiler", Offset: 0, Length: 5},
			},
			want: "hello world",
		},
		{
			name: "literal marker outside any entity is escaped",
			text: "hello * world",
			entities: []model.MessageEntity{
				{Type: "bold", Offset: 0, Length: 5},
			},
			want: "*hello* *" + zwnj + " world",
		},
		{
			name: "multi-byte text with a bold span past the ASCII prefix",
			// "Привет " is 7 runes; UTF-16 offset 7 lands right after it since
			// every rune here is in the BMP (1 UTF-16 unit each).
			text: "Привет world",
			entities: []model.MessageEntity{
				{Type: "bold", Offset: 7, Length: 5},
			},
			want: "Привет *world*",
		},
		{
			name: "entity past a supplementary-plane rune counts it as two UTF-16 units",
			// "😀" (U+1F600) is outside the BMP: 1 rune / 4 UTF-8 bytes / 2 UTF-16
			// units. "bold" here starts at UTF-16 offset 3 (1 for "a" + 2 for the
			// emoji), which must land right after the emoji's 4 UTF-8 bytes, not
			// 3 bytes in (which would split the rune).
			text: "a😀bold",
			entities: []model.MessageEntity{
				{Type: "bold", Offset: 3, Length: 4},
			},
			want: "a😀*bold*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseTelegramMarkdown(tt.text, tt.entities)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUTF16SpanToByteSpan(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		offset, length int64
		wantByteOffset int
		wantByteLength int
		wantOK         bool
	}{
		{
			name: "ascii", text: "hello world",
			offset: 0, length: 5,
			wantByteOffset: 0, wantByteLength: 5, wantOK: true,
		},
		{
			name: "ascii mid-string", text: "hello world",
			offset: 6, length: 5,
			wantByteOffset: 6, wantByteLength: 5, wantOK: true,
		},
		{
			name: "span reaching the end of the string", text: "hello",
			offset: 0, length: 5,
			wantByteOffset: 0, wantByteLength: 5, wantOK: true,
		},
		{
			name: "negative offset is rejected", text: "hello",
			offset: -1, length: 3,
			wantOK: false,
		},
		{
			name: "zero length is rejected", text: "hello",
			offset: 0, length: 0,
			wantOK: false,
		},
		{
			name: "span past the end of the string is rejected", text: "hello",
			offset: 0, length: 10,
			wantOK: false,
		},
		{
			name: "supplementary-plane rune counts as two UTF-16 units", text: "a😀bold",
			offset: 3, length: 4,
			wantByteOffset: 5, wantByteLength: 4, wantOK: true, // "a" (1 byte) + "😀" (4 bytes) = byte 5
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOffset, gotLength, gotOK := utf16SpanToByteSpan(tt.text, tt.offset, tt.length)
			if gotOK != tt.wantOK {
				t.Fatalf("ok: got %v, want %v", gotOK, tt.wantOK)
			}
			if !gotOK {
				return
			}
			if gotOffset != tt.wantByteOffset || gotLength != tt.wantByteLength {
				t.Errorf("got (offset=%d, length=%d), want (offset=%d, length=%d)", gotOffset, gotLength, tt.wantByteOffset, tt.wantByteLength)
			}
		})
	}
}
