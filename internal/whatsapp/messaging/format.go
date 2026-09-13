package messaging

import (
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/im-providers-service/internal/provider/format"
)

// RenderWhatsApp renders text with entities into WhatsApp Cloud API format (see
// developers.facebook.com/docs/whatsapp/cloud-api/messages/text-messages):
// *bold*, _italic_, ~strikethrough~, and ```monospace```. LINK entities render
// as "label (url)" relying on WhatsApp's client-side auto-linking. Degrades
// unsupported entity types and invalid LINK schemes to plain text.
func RenderWhatsApp(text string, entities []sharedmodel.Entity) string {
	return format.RenderMarkdownStyle(text, entities)
}
