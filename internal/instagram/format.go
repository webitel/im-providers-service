package instagram

import (
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/im-providers-service/internal/provider/format"
)

// RenderInstagram renders text with entities for the Instagram Messaging API
// (developers.facebook.com/docs/instagram-platform/instagram-api-with-instagram-login/messaging-api),
// which documents only a plain UTF-8 text field (<=1000 bytes) with no
// markdown/rich-text syntax, so entities degrade to plain text.
func RenderInstagram(text string, entities []sharedmodel.Entity) string {
	return format.RenderPlainDegrade(text, entities)
}

// ParseInstagramMarkdown converts an inbound Instagram DM's text into this
// service's shared markdown dialect (format.RenderMarkdownStyle's *bold*,
// _italic_, ~strikethrough~, ```monospace``` -- see RenderWhatsApp/RenderViber)
// for Body. Instagram's webhook carries plain text with no formatting
// structure, so any literal *, _, ~, or backtick a user happened to type is
// escaped rather than rendered -- otherwise it would be misread as
// intentional formatting once Body is treated as markdown downstream.
func ParseInstagramMarkdown(text string) string {
	return format.EscapeMarkdownLiteralMarkers(text)
}
