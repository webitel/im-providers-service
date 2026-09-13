package format

import (
	"sort"
	"strings"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

// RenderPlainDegrade is the shared low-level renderer that degrades text with
// entities to plain, unformatted text -- used by Facebook Messenger and
// Instagram Messaging (via their own RenderFacebook / RenderInstagram entry
// points in internal/facebook and internal/instagram), neither of which
// documents any inline text markup in its Send API. Every entity type but LINK
// simply vanishes (its span's text is already part of text and needs no
// wrapping); a LINK entity keeps its target visible as "label (url)" (matching
// WhatsApp's treatment) rather than silently dropping the URL, since neither
// platform auto-links raw URL substrings inside a plain message the way
// WhatsApp's client does.
func RenderPlainDegrade(text string, entities []sharedmodel.Entity) string {
	if len(entities) == 0 {
		return text
	}

	valid, _ := FilterValidEntities(text, entities)
	if len(valid) == 0 {
		return text
	}

	links := make([]sharedmodel.Entity, 0, len(valid))
	for _, e := range valid {
		if e.Type == TypeLink && IsValidLinkScheme(e.Value) {
			links = append(links, e)
		}
	}
	if len(links) == 0 {
		return text
	}

	sort.SliceStable(links, func(i, j int) bool {
		if links[i].Offset == links[j].Offset {
			return links[i].Length > links[j].Length
		}
		return links[i].Offset < links[j].Offset
	})
	links = DropCrossingOverlaps(links)

	var b strings.Builder
	pos := 0
	for _, e := range links {
		start, end := int(e.Offset), int(e.Offset)+int(e.Length)
		if start < pos {
			// A defensive skip: DropCrossingOverlaps only removes partial
			// (crossing) overlaps, so a link fully contained in a previous one
			// (which shouldn't occur among same-type LINK spans) is ignored
			// here rather than corrupting the output.
			continue
		}

		b.WriteString(text[pos:start])
		label := text[start:end]
		b.WriteString(label)
		if label != e.Value {
			b.WriteString(" (")
			b.WriteString(e.Value)
			b.WriteString(")")
		}
		pos = end
	}
	b.WriteString(text[pos:])

	return b.String()
}
