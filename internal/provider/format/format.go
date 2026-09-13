package format

import (
	"net/url"
	"strings"
	"unicode/utf8"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

// Type constants for allowed entity types.
const (
	TypeBold          = "BOLD"
	TypeItalic        = "ITALIC"
	TypeStrikethrough = "STRIKETHROUGH"
	TypeCode          = "CODE"
	TypePre           = "PRE"
	TypeLink          = "LINK"
)

// maxEntities mirrors the protovalidate `max_items: 100` constraint already
// declared on the wire (ProviderSendTextRequest.entities / ProviderSendDocumentRequest.entities).
// No protovalidate interceptor currently runs for this RPC (see docs/knowledge/gotchas.md),
// so this is defense-in-depth for the same documented limit.
const maxEntities = 100

// FilterValidEntities partitions entities into valid and invalid. An entity is
// invalid when its Offset is negative, its Length isn't positive, Offset+Length
// exceeds the byte length of text, either boundary doesn't align to a UTF-8 rune
// start (a mis-aligned offset -- e.g. one computed in UTF-16 code units by a
// producer -- would otherwise slice through the middle of a multi-byte rune and
// emit invalid UTF-8), or the entity is beyond the 100th (mirrors the wire's
// max_items limit). Input order is preserved in both output slices.
func FilterValidEntities(text string, entities []sharedmodel.Entity) (valid, invalid []sharedmodel.Entity) {
	textLen := len(text)
	valid = make([]sharedmodel.Entity, 0, len(entities))
	invalid = make([]sharedmodel.Entity, 0, len(entities))

	for i, e := range entities {
		if i >= maxEntities {
			invalid = append(invalid, e)
			continue
		}

		off, length := int(e.Offset), int(e.Length)
		if e.Offset < 0 || e.Length <= 0 || off+length > textLen {
			invalid = append(invalid, e)
			continue
		}
		if !utf8.RuneStart(text[off]) || (off+length < textLen && !utf8.RuneStart(text[off+length])) {
			invalid = append(invalid, e)
			continue
		}

		valid = append(valid, e)
	}

	return valid, invalid
}

// IsAllowedType returns true if the entity type is in the allow-list.
func IsAllowedType(t string) bool {
	switch t {
	case TypeBold, TypeItalic, TypeStrikethrough, TypeCode, TypePre, TypeLink:
		return true
	default:
		return false
	}
}

// IsValidLinkScheme reports whether urlStr is an http, https, or mailto URL that
// actually names a target -- a bare scheme like "https:" or "mailto:" is rejected
// even though it parses without error.
func IsValidLinkScheme(urlStr string) bool {
	u, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return u.Host != ""
	case "mailto":
		return u.Opaque != "" || u.Path != ""
	default:
		return false
	}
}

// DropCrossingOverlaps removes the later-starting entity of any pair that
// partially overlaps (crosses) without one containing the other. Entity a
// crosses b when a.Offset < b.Offset && b.Offset < a.Offset+a.Length &&
// a.Offset+a.Length < b.Offset+b.Length -- in that case b (the later-starting
// entity) is dropped, defensively, so markers never interleave invalidly.
func DropCrossingOverlaps(entities []sharedmodel.Entity) []sharedmodel.Entity {
	dropped := make([]bool, len(entities))
	for i, a := range entities {
		if dropped[i] {
			continue
		}
		for j, b := range entities {
			if i == j || dropped[j] {
				continue
			}
			if a.Offset < b.Offset && b.Offset < a.Offset+a.Length && a.Offset+a.Length < b.Offset+b.Length {
				dropped[j] = true
			}
		}
	}

	result := make([]sharedmodel.Entity, 0, len(entities))
	for i, e := range entities {
		if !dropped[i] {
			result = append(result, e)
		}
	}
	return result
}

// DropNestedInOpaque removes any entity strictly contained within a CODE, PRE, or
// LINK entity (including an entity sharing the exact same span as one of those).
// Telegram's HTML parse_mode rejects nested tags inside <pre>/<code> and a nested
// <a> inside <a>; applied uniformly across renderers so WhatsApp and Telegram
// produce consistent output for the same input.
func DropNestedInOpaque(entities []sharedmodel.Entity) []sharedmodel.Entity {
	isOpaque := func(t string) bool { return t == TypeCode || t == TypePre || t == TypeLink }

	dropped := make([]bool, len(entities))
	for i, outer := range entities {
		if dropped[i] || !isOpaque(outer.Type) {
			continue
		}
		for j, inner := range entities {
			if i == j || dropped[j] {
				continue
			}
			if inner.Offset >= outer.Offset && inner.Offset+inner.Length <= outer.Offset+outer.Length {
				dropped[j] = true
			}
		}
	}

	result := make([]sharedmodel.Entity, 0, len(entities))
	for i, e := range entities {
		if !dropped[i] {
			result = append(result, e)
		}
	}
	return result
}
