package format

import (
	"sort"
	"strings"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

// RenderMarkdownStyle is the shared low-level renderer for the inline markdown
// syntax documented, independently, by both WhatsApp's Cloud API
// (developers.facebook.com/docs/whatsapp/cloud-api/messages/text-messages) and
// Viber's Bot API (developers.viber.com/docs/tools/text-formatting): *bold*,
// _italic_, ~strikethrough~, and ```monospace```. LINK entities render as
// "label (url)" (neither platform's docs describe dedicated link markup; the
// label is omitted from the parenthetical when it already is the URL).
// Literal formatting characters that sit at a word-boundary position in plain
// (non-entity-owned) text are escaped with U+200C to prevent accidental
// formatting triggers -- both platforms require a marker to sit hard against
// non-space content to open/close a span. Degrades unsupported entity types and
// invalid LINK schemes to plain text.
//
// Each of the two platforms that use this syntax owns its own RenderWhatsApp /
// RenderViber entry point (in internal/whatsapp/messaging and internal/viber
// respectively), which simply wraps this function with a doc comment citing
// that platform's own docs -- the syntax genuinely is identical, verified
// independently against each platform's reference, not an assumption.
func RenderMarkdownStyle(text string, entities []sharedmodel.Entity) string {
	if len(entities) == 0 {
		return EscapeMarkdownLiteralMarkers(text)
	}

	valid, _ := FilterValidEntities(text, entities)
	if len(valid) == 0 {
		return EscapeMarkdownLiteralMarkers(text)
	}

	// Keep only allowed types, validate LINK schemes.
	filtered := make([]sharedmodel.Entity, 0, len(valid))
	for _, e := range valid {
		if !IsAllowedType(e.Type) {
			continue
		}
		if e.Type == TypeLink && !IsValidLinkScheme(e.Value) {
			continue
		}
		filtered = append(filtered, e)
	}
	if len(filtered) == 0 {
		return EscapeMarkdownLiteralMarkers(text)
	}

	// Sort by offset ascending, then by length descending (for proper nesting).
	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].Offset == filtered[j].Offset {
			return filtered[i].Length > filtered[j].Length
		}
		return filtered[i].Offset < filtered[j].Offset
	})

	filtered = DropNestedInOpaque(filtered)
	filtered = DropCrossingOverlaps(filtered)

	var result strings.Builder
	type openEntity struct {
		entity sharedmodel.Entity
	}
	var stack []openEntity
	pos := 0

	closeEntity := func(e sharedmodel.Entity) {
		if e.Type == TypeLink {
			label := text[e.Offset : int(e.Offset)+int(e.Length)]
			if label != e.Value {
				result.WriteString(" (")
				result.WriteString(e.Value)
				result.WriteString(")")
			}
			return
		}
		result.WriteString(markdownMarker(e.Type))
	}

	for pos < len(text) {
		// Close entities that end at this position.
		for len(stack) > 0 && pos == int(stack[len(stack)-1].entity.Offset)+int(stack[len(stack)-1].entity.Length) {
			open := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			closeEntity(open.entity)
		}

		// Open entities that start at this position.
		for _, e := range filtered {
			if int(e.Offset) == pos {
				stack = append(stack, openEntity{entity: e})
				if e.Type != TypeLink {
					result.WriteString(markdownMarker(e.Type))
				}
			}
		}

		// Find next boundary (entity start/end or text end).
		nextBoundary := len(text)
		for _, e := range filtered {
			eStart := int(e.Offset)
			eEnd := int(e.Offset) + int(e.Length)
			if eStart > pos {
				nextBoundary = min(nextBoundary, eStart)
			}
			if eEnd > pos && eEnd < nextBoundary {
				nextBoundary = eEnd
			}
		}

		// Literal text bytes not owned by any open entity get marker escaping;
		// bytes inside an open entity (e.g. a CODE/PRE span, or a LINK label)
		// are copied verbatim so a recipient's copy-paste isn't corrupted.
		segment := text[pos:nextBoundary]
		if len(stack) == 0 {
			result.WriteString(EscapeMarkdownLiteralMarkers(segment))
		} else {
			result.WriteString(segment)
		}
		pos = nextBoundary
	}

	// Close any remaining open entities.
	for len(stack) > 0 {
		open := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		closeEntity(open.entity)
	}

	return result.String()
}

// markdownMarker returns the shared WhatsApp/Viber delimiter for a basic
// (non-LINK) entity type -- both platforms use the identical character as
// opening and closing marker for every type in this syntax.
func markdownMarker(t string) string {
	switch t {
	case TypeBold:
		return "*"
	case TypeItalic:
		return "_"
	case TypeStrikethrough:
		return "~"
	case TypeCode, TypePre:
		return "```"
	default:
		return ""
	}
}

// EscapeMarkdownLiteralMarkers inserts a zero-width non-joiner (U+200C) immediately
// adjacent to a literal *, _, ~, or backtick character that sits at a position
// this markdown-like parser treats as a word boundary (start/end of the segment,
// or adjacent to whitespace) -- the only positions where such a character can pair
// up with another to trigger unintended formatting. A marker strictly between two
// non-space characters (e.g. inside a URL like ".../a_b_c" or an expression like
// "2*3") is left untouched, since the parser itself would never treat it as a
// formatting delimiter there.
func EscapeMarkdownLiteralMarkers(s string) string {
	const zeroWidthNonJoiner = "‌"

	var b strings.Builder
	n := len(s)
	for i := 0; i < n; i++ {
		c := s[i]
		b.WriteByte(c)
		if !isMarkdownMarkerByte(c) {
			continue
		}
		leftBoundary := i == 0 || isASCIISpace(s[i-1])
		rightBoundary := i == n-1 || isASCIISpace(s[i+1])
		if leftBoundary || rightBoundary {
			b.WriteString(zeroWidthNonJoiner)
		}
	}
	return b.String()
}

func isMarkdownMarkerByte(c byte) bool {
	return c == '*' || c == '_' || c == '~' || c == '`'
}

func isASCIISpace(c byte) bool {
	switch c {
	case ' ', '\t', '\n', '\r', '\v', '\f':
		return true
	default:
		return false
	}
}
