package format

import (
	"testing"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

const zwnj = "‌"

func TestFilterValidEntities(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		entities       []sharedmodel.Entity
		wantValidLen   int
		wantInvalidLen int
	}{
		{
			name:           "all valid",
			text:           "hello world",
			entities:       []sharedmodel.Entity{{Type: "BOLD", Offset: 0, Length: 5}},
			wantValidLen:   1,
			wantInvalidLen: 0,
		},
		{
			name:           "negative offset",
			text:           "hello",
			entities:       []sharedmodel.Entity{{Type: "BOLD", Offset: -1, Length: 3}},
			wantValidLen:   0,
			wantInvalidLen: 1,
		},
		{
			name:           "zero length",
			text:           "hello",
			entities:       []sharedmodel.Entity{{Type: "BOLD", Offset: 0, Length: 0}},
			wantValidLen:   0,
			wantInvalidLen: 1,
		},
		{
			name:           "offset+length exceeds text",
			text:           "hello",
			entities:       []sharedmodel.Entity{{Type: "BOLD", Offset: 3, Length: 5}},
			wantValidLen:   0,
			wantInvalidLen: 1,
		},
		{
			name:           "exact boundary",
			text:           "hello",
			entities:       []sharedmodel.Entity{{Type: "BOLD", Offset: 0, Length: 5}},
			wantValidLen:   1,
			wantInvalidLen: 0,
		},
		{
			name: "mixed valid and invalid",
			text: "hello world",
			entities: []sharedmodel.Entity{
				{Type: "BOLD", Offset: 0, Length: 5},
				{Type: "ITALIC", Offset: 100, Length: 5},
				{Type: "STRIKETHROUGH", Offset: 6, Length: 5},
			},
			wantValidLen:   2,
			wantInvalidLen: 1,
		},
		{
			// "Привет" is 6 runes / 12 bytes ("П" etc. are 2 bytes each in UTF-8).
			// Offset 1 lands mid-rune (inside the first byte pair), which a
			// UTF-16-based producer could plausibly send.
			name:           "offset misaligned with rune boundary",
			text:           "Привет",
			entities:       []sharedmodel.Entity{{Type: "BOLD", Offset: 1, Length: 4}},
			wantValidLen:   0,
			wantInvalidLen: 1,
		},
		{
			name:           "end misaligned with rune boundary",
			text:           "Привет",
			entities:       []sharedmodel.Entity{{Type: "BOLD", Offset: 0, Length: 3}},
			wantValidLen:   0,
			wantInvalidLen: 1,
		},
		{
			name:           "rune-aligned offsets on multi-byte text",
			text:           "Привет",
			entities:       []sharedmodel.Entity{{Type: "BOLD", Offset: 0, Length: 4}},
			wantValidLen:   1,
			wantInvalidLen: 0,
		},
		{
			name: "more than max entities",
			text: "0123456789",
			entities: func() []sharedmodel.Entity {
				es := make([]sharedmodel.Entity, 0, maxEntities+5)
				for i := 0; i < maxEntities+5; i++ {
					es = append(es, sharedmodel.Entity{Type: "BOLD", Offset: 0, Length: 1})
				}
				return es
			}(),
			wantValidLen:   maxEntities,
			wantInvalidLen: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, invalid := FilterValidEntities(tt.text, tt.entities)
			if len(valid) != tt.wantValidLen {
				t.Errorf("valid length: got %d, want %d", len(valid), tt.wantValidLen)
			}
			if len(invalid) != tt.wantInvalidLen {
				t.Errorf("invalid length: got %d, want %d", len(invalid), tt.wantInvalidLen)
			}
		})
	}
}

func TestRenderMarkdownStyle(t *testing.T) {
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
				{Type: TypeBold, Offset: 0, Length: 5},
			},
			want: "*hello* world",
		},
		{
			name: "italic",
			text: "hello world",
			entities: []sharedmodel.Entity{
				{Type: TypeItalic, Offset: 6, Length: 5},
			},
			want: "hello _world_",
		},
		{
			name: "strikethrough",
			text: "hello world",
			entities: []sharedmodel.Entity{
				{Type: TypeStrikethrough, Offset: 0, Length: 5},
			},
			want: "~hello~ world",
		},
		{
			name: "code",
			text: "hello world",
			entities: []sharedmodel.Entity{
				{Type: TypeCode, Offset: 0, Length: 5},
			},
			want: "```hello``` world",
		},
		{
			name: "pre",
			text: "hello world",
			entities: []sharedmodel.Entity{
				{Type: TypePre, Offset: 0, Length: 5},
			},
			want: "```hello``` world",
		},
		{
			name: "link with valid https",
			text: "click here",
			entities: []sharedmodel.Entity{
				{Type: TypeLink, Offset: 6, Length: 4, Value: "https://example.com"},
			},
			want: "click here (https://example.com)",
		},
		{
			name: "link with valid http",
			text: "click here",
			entities: []sharedmodel.Entity{
				{Type: TypeLink, Offset: 6, Length: 4, Value: "http://example.com"},
			},
			want: "click here (http://example.com)",
		},
		{
			name: "link with mailto",
			text: "email me",
			entities: []sharedmodel.Entity{
				{Type: TypeLink, Offset: 0, Length: 5, Value: "mailto:test@example.com"},
			},
			want: "email (mailto:test@example.com) me",
		},
		{
			name: "link whose label is already the URL is not duplicated",
			text: "https://example.com",
			entities: []sharedmodel.Entity{
				{Type: TypeLink, Offset: 0, Length: 20, Value: "https://example.com"},
			},
			want: "https://example.com",
		},
		{
			name: "link with invalid scheme",
			text: "click here",
			entities: []sharedmodel.Entity{
				{Type: TypeLink, Offset: 6, Length: 4, Value: "javascript:alert(1)"},
			},
			want: "click here",
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
				{Type: TypeBold, Offset: 0, Length: 10},
			},
			want: "hello",
		},
		{
			name: "multiple entities",
			text: "hello world foo",
			entities: []sharedmodel.Entity{
				{Type: TypeBold, Offset: 0, Length: 5},
				{Type: TypeItalic, Offset: 6, Length: 5},
				{Type: TypeStrikethrough, Offset: 12, Length: 3},
			},
			want: "*hello* _world_ ~foo~",
		},
		{
			name:     "no entities: markers inside a URL are left untouched",
			text:     "see https://ex.com/a_b_c and 2*3",
			entities: nil,
			want:     "see https://ex.com/a_b_c and 2*3",
		},
		{
			name:     "no entities: a marker at a word boundary is escaped",
			text:     "hello * world",
			entities: nil,
			want:     "hello *" + zwnj + " world",
		},
		{
			name:     "no entities: marker at start of string is escaped",
			text:     "*hello world",
			entities: nil,
			want:     "*" + zwnj + "hello world",
		},
		{
			name:     "no entities: marker at end of string is escaped",
			text:     "hello world*",
			entities: nil,
			want:     "hello world*" + zwnj,
		},
		{
			name: "plain-run underscores around an entity stay untouched, entity content is not escaped",
			text: "a_b bold c_d",
			entities: []sharedmodel.Entity{
				{Type: TypeBold, Offset: 4, Length: 4},
			},
			// "a_b " and " c_d" are plain runs: their `_` sit between letters (not
			// boundary-adjacent) so stay untouched, and the BOLD span's own
			// asterisks are the structural markers (not ZWNJ-escaped, since
			// they're not in a plain run).
			want: "a_b *bold* c_d",
		},
		{
			name: "markers inside a CODE span are not escaped (copy-paste safety)",
			text: "run foo_bar()",
			entities: []sharedmodel.Entity{
				{Type: TypeCode, Offset: 4, Length: 9},
			},
			want: "run ```foo_bar()```",
		},
		{
			name: "bold nested inside code is dropped",
			text: "code",
			entities: []sharedmodel.Entity{
				{Type: TypeCode, Offset: 0, Length: 4},
				{Type: TypeBold, Offset: 0, Length: 4},
			},
			want: "```code```",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderMarkdownStyle(tt.text, tt.entities)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderPlainDegrade(t *testing.T) {
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
			name: "non-link entities are ignored",
			text: "hello world",
			entities: []sharedmodel.Entity{
				{Type: TypeBold, Offset: 0, Length: 5},
				{Type: TypeItalic, Offset: 6, Length: 5},
			},
			want: "hello world",
		},
		{
			name: "unrecognized entity type degrades to plain text",
			text: "hello world",
			entities: []sharedmodel.Entity{
				{Type: "SPOILER", Offset: 0, Length: 5},
			},
			want: "hello world",
		},
		{
			name: "empty text",
			text: "",
			entities: []sharedmodel.Entity{
				{Type: TypeBold, Offset: 0, Length: 0},
			},
			want: "",
		},
		{
			name: "link keeps its URL visible as a parenthetical",
			text: "click here",
			entities: []sharedmodel.Entity{
				{Type: TypeLink, Offset: 6, Length: 4, Value: "https://example.com"},
			},
			want: "click here (https://example.com)",
		},
		{
			name: "link with invalid scheme degrades to plain label",
			text: "click here",
			entities: []sharedmodel.Entity{
				{Type: TypeLink, Offset: 6, Length: 4, Value: "javascript:alert(1)"},
			},
			want: "click here",
		},
		{
			name: "link whose label is already the URL is not duplicated",
			text: "https://example.com",
			entities: []sharedmodel.Entity{
				{Type: TypeLink, Offset: 0, Length: 20, Value: "https://example.com"},
			},
			want: "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderPlainDegrade(tt.text, tt.entities)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsValidLinkScheme(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://example.com", true},
		{"http://example.com", true},
		{"mailto:test@example.com", true},
		{"HTTPS://EXAMPLE.COM", true},
		{"javascript:alert(1)", false},
		{"data:text/html,<script>alert(1)</script>", false},
		{"ftp://example.com", false},
		{"https:", false},
		{"http:/x", false},
		{"mailto:", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := IsValidLinkScheme(tt.url)
			if got != tt.want {
				t.Errorf("IsValidLinkScheme(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestDropCrossingOverlaps(t *testing.T) {
	tests := []struct {
		name     string
		entities []sharedmodel.Entity
		want     []sharedmodel.Entity
	}{
		{
			name: "no overlaps",
			entities: []sharedmodel.Entity{
				{Type: TypeBold, Offset: 0, Length: 5},
				{Type: TypeItalic, Offset: 6, Length: 5},
			},
			want: []sharedmodel.Entity{
				{Type: TypeBold, Offset: 0, Length: 5},
				{Type: TypeItalic, Offset: 6, Length: 5},
			},
		},
		{
			name: "one contains another",
			entities: []sharedmodel.Entity{
				{Type: TypeBold, Offset: 0, Length: 10},
				{Type: TypeItalic, Offset: 2, Length: 3},
			},
			want: []sharedmodel.Entity{
				{Type: TypeBold, Offset: 0, Length: 10},
				{Type: TypeItalic, Offset: 2, Length: 3},
			},
		},
		{
			name: "crossing overlap drops the later-starting entity",
			entities: []sharedmodel.Entity{
				{Type: TypeBold, Offset: 0, Length: 5},
				{Type: TypeItalic, Offset: 3, Length: 5},
			},
			want: []sharedmodel.Entity{
				{Type: TypeBold, Offset: 0, Length: 5},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DropCrossingOverlaps(tt.entities)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d entities after dropping overlaps, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("entity %d: got %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestDropNestedInOpaque(t *testing.T) {
	tests := []struct {
		name     string
		entities []sharedmodel.Entity
		want     []sharedmodel.Entity
	}{
		{
			name: "non-opaque containment is left alone",
			entities: []sharedmodel.Entity{
				{Type: TypeBold, Offset: 0, Length: 10},
				{Type: TypeItalic, Offset: 2, Length: 3},
			},
			want: []sharedmodel.Entity{
				{Type: TypeBold, Offset: 0, Length: 10},
				{Type: TypeItalic, Offset: 2, Length: 3},
			},
		},
		{
			name: "entity contained in a CODE span is dropped",
			entities: []sharedmodel.Entity{
				{Type: TypeCode, Offset: 0, Length: 10},
				{Type: TypeBold, Offset: 2, Length: 3},
			},
			want: []sharedmodel.Entity{
				{Type: TypeCode, Offset: 0, Length: 10},
			},
		},
		{
			name: "same-span entity sharing a PRE span is dropped",
			entities: []sharedmodel.Entity{
				{Type: TypePre, Offset: 0, Length: 4},
				{Type: TypeBold, Offset: 0, Length: 4},
			},
			want: []sharedmodel.Entity{
				{Type: TypePre, Offset: 0, Length: 4},
			},
		},
		{
			name: "link nested in another link is dropped",
			entities: []sharedmodel.Entity{
				{Type: TypeLink, Offset: 0, Length: 10, Value: "https://outer.example"},
				{Type: TypeLink, Offset: 2, Length: 4, Value: "https://inner.example"},
			},
			want: []sharedmodel.Entity{
				{Type: TypeLink, Offset: 0, Length: 10, Value: "https://outer.example"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DropNestedInOpaque(tt.entities)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d entities, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("entity %d: got %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestEscapeMarkdownLiteralMarkers(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "marker surrounded by spaces is escaped",
			input: "hello * world",
			want:  "hello *" + zwnj + " world",
		},
		{
			name:  "underscore surrounded by spaces is escaped",
			input: "hello _ world",
			want:  "hello _" + zwnj + " world",
		},
		{
			name:  "tilde surrounded by spaces is escaped",
			input: "hello ~ world",
			want:  "hello ~" + zwnj + " world",
		},
		{
			name:  "backtick surrounded by spaces is escaped",
			input: "hello ` world",
			want:  "hello `" + zwnj + " world",
		},
		{
			name:  "marker between two letters is left untouched",
			input: "a_b_c",
			want:  "a_b_c",
		},
		{
			name:  "arithmetic asterisk between digits is left untouched",
			input: "2*3",
			want:  "2*3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EscapeMarkdownLiteralMarkers(tt.input)
			if got != tt.want {
				t.Errorf("EscapeMarkdownLiteralMarkers(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
