package webhook

// ParseWhatsAppMarkdown converts an inbound WhatsApp message's text (or media
// caption) into this service's shared markdown dialect for Body/Document.Body/
// Image.Body. WhatsApp's Cloud API webhook already hands back the literal
// *bold*/_italic_/~strikethrough~/```monospace``` markup a user typed
// client-side (developers.facebook.com/docs/whatsapp/cloud-api/messages/text-messages),
// which is exactly the dialect format.RenderMarkdownStyle produces for the
// outbound direction (see internal/whatsapp/messaging.RenderWhatsApp), so no
// conversion is needed here.
func ParseWhatsAppMarkdown(text string) string {
	return text
}
