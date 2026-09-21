package bot

import (
	"sort"
	"strings"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/im-providers-service/internal/provider/format"
	"github.com/webitel/im-providers-service/internal/telegram/bot/model"
)

// ParseTelegramMarkdown converts a Telegram message's text and entities
// (core.telegram.org/bots/api#messageentity) into the shared markdown dialect
// used across this service's Body field: *bold*, _italic_, ~strikethrough~,
// and ```monospace``` (format.RenderMarkdownStyle -- the same renderer
// internal/whatsapp/messaging and internal/viber use on the outbound side, so
// text relayed from Telegram lines up with what WhatsApp/Viber already send
// natively). Only entity types with an equivalent in that dialect (bold,
// italic, strikethrough, code, pre, text_link) are kept; everything else
// (mentions, hashtags, spoilers, custom emoji, ...) degrades to plain text.
// Telegram entity offsets/lengths are in UTF-16 code units, not bytes, so each
// entity is converted to a UTF-8 byte span before rendering; an entity that
// fails to convert (e.g. a malformed offset) is dropped rather than
// mis-slicing the message.
func ParseTelegramMarkdown(text string, entities []model.MessageEntity) string {
	if len(entities) == 0 {
		return format.EscapeMarkdownLiteralMarkers(text)
	}

	converted := make([]sharedmodel.Entity, 0, len(entities))
	for _, e := range entities {
		typ, ok := markdownTypeForTelegramEntity(e.Type)
		if !ok {
			continue
		}

		byteOffset, byteLength, ok := utf16SpanToByteSpan(text, e.Offset, e.Length)
		if !ok {
			continue
		}

		value := ""
		if typ == format.TypeLink {
			if e.URL == nil || *e.URL == "" {
				continue
			}
			value = *e.URL
		}

		converted = append(converted, sharedmodel.Entity{
			Type:   typ,
			Offset: int32(byteOffset),
			Length: int32(byteLength),
			Value:  value,
		})
	}

	return format.RenderMarkdownStyle(text, converted)
}

// markdownTypeForTelegramEntity maps a Telegram MessageEntity.Type to this
// service's canonical entity type. "text_link" maps to LINK; entity types
// with no equivalent in the shared markdown dialect (mention, hashtag,
// cashtag, bot_command, url, email, phone_number, underline, spoiler,
// blockquote, expandable_blockquote, text_mention, custom_emoji, date_time)
// report ok=false so the caller lets that span degrade to plain text.
func markdownTypeForTelegramEntity(telegramType string) (t string, ok bool) {
	switch telegramType {
	case "bold":
		return format.TypeBold, true
	case "italic":
		return format.TypeItalic, true
	case "strikethrough":
		return format.TypeStrikethrough, true
	case "code":
		return format.TypeCode, true
	case "pre":
		return format.TypePre, true
	case "text_link":
		return format.TypeLink, true
	default:
		return "", false
	}
}

// utf16SpanToByteSpan converts a [utf16Offset, utf16Offset+utf16Length) span,
// expressed in UTF-16 code units as Telegram's Bot API defines it, into a byte
// offset/length within the UTF-8-encoded text. Returns ok=false if either
// boundary is negative or falls outside the string (a malformed or
// UTF-8-code-unit span a buggy/malicious producer could send).
func utf16SpanToByteSpan(text string, utf16Offset, utf16Length int64) (byteOffset, byteLength int, ok bool) {
	if utf16Offset < 0 || utf16Length <= 0 {
		return 0, 0, false
	}

	targetStart := utf16Offset
	targetEnd := utf16Offset + utf16Length

	units := int64(0)
	startByte, endByte := -1, -1

	for i, r := range text {
		if units == targetStart {
			startByte = i
		}
		if units == targetEnd {
			endByte = i
		}
		if startByte != -1 && endByte != -1 {
			break
		}

		if r > 0xFFFF {
			units += 2 // outside the BMP: encoded as a UTF-16 surrogate pair
		} else {
			units++
		}
	}

	if startByte == -1 && units == targetStart {
		startByte = len(text)
	}
	if endByte == -1 && units == targetEnd {
		endByte = len(text)
	}
	if startByte == -1 || endByte == -1 || endByte < startByte {
		return 0, 0, false
	}

	return startByte, endByte - startByte, true
}

// RenderTelegramHTML renders text with entities into Telegram Bot API HTML
// parse_mode format (core.telegram.org/bots/api#html-style): <b>, <i>, <s>,
// <code>, <pre>, <a href="..."> tags. Escapes &, <, >, and " in appropriate
// contexts. Degrades unsupported entity types and invalid LINK schemes to
// plain text.
func RenderTelegramHTML(text string, entities []sharedmodel.Entity) string {
	if len(entities) == 0 {
		return escapeHTMLText(text)
	}

	// Filter out-of-bounds entities.
	valid, _ := format.FilterValidEntities(text, entities)
	if len(valid) == 0 {
		return escapeHTMLText(text)
	}

	// Keep only allowed types, validate LINK schemes.
	filtered := make([]sharedmodel.Entity, 0, len(valid))
	for _, e := range valid {
		if !format.IsAllowedType(e.Type) {
			continue
		}
		if e.Type == format.TypeLink && !format.IsValidLinkScheme(e.Value) {
			continue
		}
		filtered = append(filtered, e)
	}
	if len(filtered) == 0 {
		return escapeHTMLText(text)
	}

	// Sort by offset ascending, then by length descending (for proper nesting).
	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].Offset == filtered[j].Offset {
			return filtered[i].Length > filtered[j].Length
		}
		return filtered[i].Offset < filtered[j].Offset
	})

	// Drop entities nested inside a CODE/PRE/LINK span (Telegram rejects nested
	// tags there), then drop any remaining partial (crossing) overlaps.
	filtered = format.DropNestedInOpaque(filtered)
	filtered = format.DropCrossingOverlaps(filtered)

	// Stack-based rendering.
	var result strings.Builder
	type openEntity struct {
		entity sharedmodel.Entity
	}

	var stack []openEntity
	pos := 0

	for pos < len(text) {
		// Close entities that end at this position.
		for len(stack) > 0 && pos == int(stack[len(stack)-1].entity.Offset)+int(stack[len(stack)-1].entity.Length) {
			open := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			result.WriteString(closeTag(open.entity.Type))
		}

		// Open entities that start at this position.
		for _, e := range filtered {
			if int(e.Offset) == pos {
				stack = append(stack, openEntity{entity: e})
				result.WriteString(openTag(e))
			}
		}

		// Copy literal text up to the next entity boundary.
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

		// Extract and escape literal text segment. HTML text escaping applies
		// unconditionally, including inside CODE/PRE/link-label spans, since &,
		// <, > are reserved everywhere in HTML content, not just in plain runs.
		segment := text[pos:nextBoundary]
		result.WriteString(escapeHTMLText(segment))
		pos = nextBoundary
	}

	// Close any remaining open entities.
	for len(stack) > 0 {
		open := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		result.WriteString(closeTag(open.entity.Type))
	}

	return result.String()
}

// openTag returns the opening HTML tag for an entity type.
func openTag(e sharedmodel.Entity) string {
	switch e.Type {
	case format.TypeBold:
		return "<b>"
	case format.TypeItalic:
		return "<i>"
	case format.TypeStrikethrough:
		return "<s>"
	case format.TypeCode:
		return "<code>"
	case format.TypePre:
		return "<pre>"
	case format.TypeLink:
		return `<a href="` + escapeHTMLAttr(e.Value) + `">`
	default:
		return ""
	}
}

// closeTag returns the closing HTML tag for an entity type.
func closeTag(t string) string {
	switch t {
	case format.TypeBold:
		return "</b>"
	case format.TypeItalic:
		return "</i>"
	case format.TypeStrikethrough:
		return "</s>"
	case format.TypeCode:
		return "</code>"
	case format.TypePre:
		return "</pre>"
	case format.TypeLink:
		return "</a>"
	default:
		return ""
	}
}

// escapeHTMLText escapes &, <, > for use in HTML text content.
func escapeHTMLText(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// escapeHTMLAttr escapes &, ", <, and > for use inside an HTML double-quoted
// attribute value. This is specifically for href="..." attributes since LINK's
// Value is interpolated there -- an unescaped < or > would let a crafted URL break
// out of the attribute and produce HTML Telegram's parser rejects outright.
func escapeHTMLAttr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
