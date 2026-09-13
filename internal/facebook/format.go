package facebook

import (
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/im-providers-service/internal/provider/format"
)

// RenderFacebook renders text with entities for the Messenger Platform Send API
// (developers.facebook.com/docs/messenger-platform/reference/send-api). The
// reference documents no markdown/rich-text syntax for the "text" field -- the
// asterisk/underscore/tilde/backtick rendering some users see is an undocumented
// behavior of the desktop Messenger web client, not a guaranteed API feature --
// so entities degrade to plain text.
func RenderFacebook(text string, entities []sharedmodel.Entity) string {
	return format.RenderPlainDegrade(text, entities)
}

// ParseFacebookMarkdown converts an inbound Messenger message's text into this
// service's shared markdown dialect (format.RenderMarkdownStyle's *bold*,
// _italic_, ~strikethrough~, ```monospace``` -- see RenderWhatsApp/RenderViber)
// for Body. Messenger's webhook carries plain text with no formatting
// structure, so any literal *, _, ~, or backtick a user happened to type is
// escaped rather than rendered -- otherwise it would be misread as
// intentional formatting once Body is treated as markdown downstream.
func ParseFacebookMarkdown(text string) string {
	return format.EscapeMarkdownLiteralMarkers(text)
}
