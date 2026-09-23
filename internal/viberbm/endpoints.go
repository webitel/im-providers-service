package viberbm

// InfoBip Viber Business Messages send endpoint.
// https://www.infobip.com/docs/api/channels/viber/viber-business-messages/send-viber-business-messages
const sendMessagesPath = "/viber/2/messages"

// Outbound content type discriminators.
// https://www.infobip.com/docs/api/channels/viber/viber-business-messages/outbound-message/send-viber-business-message
const (
	contentTypeText     = "TEXT"
	contentTypeImage    = "IMAGE"
	contentTypeFile     = "FILE"
	contentTypeTemplate = "TEMPLATE"
)

// authScheme is the InfoBip API-key Authorization scheme: "App {apiKey}".
// https://www.infobip.com/docs/essentials/api-authentication#api-key-header
const authScheme = "App"
