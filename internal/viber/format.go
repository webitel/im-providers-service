package viber

import (
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/im-providers-service/internal/provider/format"
)

// RenderViber renders text with entities into Viber Bot API format. Viber's own
// docs (developers.viber.com/docs/tools/text-formatting) document the same
// *bold*, _italic_, ~strikethrough~, and ```monospace``` marker set as
// WhatsApp's Cloud API, with the identical word-boundary requirement (a marker
// must sit hard against non-space content to open/close a span -- e.g. "*Viber*
// bots" formats but "*Viber*bots" doesn't); only Viber 14.6+ renders the
// formatting, older clients show the literal markers. Viber's docs describe no
// dedicated link markup, so LINK entities render as "label (url)", matching
// WhatsApp's treatment. Degrades unsupported entity types and invalid LINK
// schemes to plain text.
func RenderViber(text string, entities []sharedmodel.Entity) string {
	return format.RenderMarkdownStyle(text, entities)
}

// ParseViberMarkdown converts an inbound Viber message's text into this
// service's shared markdown dialect for Body. Viber's own docs say a message a
// user formatted client-side "will be displayed with the markdown symbols in
// the callback text parameter" -- i.e. Viber's webhook already hands back the
// literal *bold*/_italic_/~strikethrough~/```monospace``` markup, in exactly
// the dialect format.RenderMarkdownStyle produces for the outbound direction
// (see RenderViber above), so no conversion is needed here.
func ParseViberMarkdown(text string) string {
	return text
}
