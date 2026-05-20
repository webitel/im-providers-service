package facebook

import "net/url"

// Inbound webhook payload types.
// https://developers.facebook.com/docs/messenger-platform/reference/webhook-events

// WebhookRequest is the top-level payload for all Messenger webhook events.
type WebhookRequest struct {
	Object string  `json:"object"`
	Entry  []Entry `json:"entry"`
}

type Entry struct {
	ID        string      `json:"id"`
	Time      int64       `json:"time"`
	Messaging []Messaging `json:"messaging"`
}

type Messaging struct {
	Sender    Actor           `json:"sender"`
	Recipient Actor           `json:"recipient"`
	Timestamp int64           `json:"timestamp"`
	Message   *InboundMessage `json:"message,omitempty"`
	Postback  *Postback       `json:"postback,omitempty"`
}

type Actor struct {
	ID string `json:"id"`
}

type InboundMessage struct {
	Mid         string       `json:"mid"`
	Text        string       `json:"text,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

type Attachment struct {
	Type    string            `json:"type"`
	Payload AttachmentPayload `json:"payload"`
}

type AttachmentPayload struct {
	URL   string `json:"url,omitempty"`
	Title string `json:"title,omitempty"`
	Name  string `json:"name,omitempty"`
}

// Postback is sent when a user taps a persistent-menu or template button.
// https://developers.facebook.com/docs/messenger-platform/reference/webhook-events/messaging-postbacks
type Postback struct {
	Title   string `json:"title"`
	Payload string `json:"payload"`
}

// AllMessages flattens all entry messaging events into a single slice.
func (r *WebhookRequest) AllMessages() []Messaging {
	total := 0
	for i := range r.Entry {
		total += len(r.Entry[i].Messaging)
	}
	out := make([]Messaging, 0, total)
	for i := range r.Entry {
		out = append(out, r.Entry[i].Messaging...)
	}
	return out
}

// UserProfile holds the fields returned by the Graph API user node.
// https://developers.facebook.com/docs/messenger-platform/identity/user-profile#fields
type UserProfile struct {
	ID         string `json:"id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	ProfilePic string `json:"profile_pic"`
	Locale     string `json:"locale"`
	Timezone   int    `json:"timezone"`
}

// VerifyRequest holds parameters for the Facebook webhook verification handshake.
// https://developers.facebook.com/docs/messenger-platform/webhooks#verification-requests
type VerifyRequest struct {
	Mode        string
	Challenge   string
	VerifyToken string
}

func parseVerify(vals url.Values) *VerifyRequest {
	return &VerifyRequest{
		Mode:        vals.Get("hub.mode"),
		Challenge:   vals.Get("hub.challenge"),
		VerifyToken: vals.Get("hub.verify_token"),
	}
}
