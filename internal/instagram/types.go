package instagram

import "net/url"

// Inbound webhook payload types.
// https://developers.instagram.com/docs/instagram-api/webhooks/reference

// Instagram-only attachment types.
const (
	AttachmentTypeStoryMention = "story_mention"
	AttachmentTypeIGReel       = "ig_reel"
	AttachmentTypeReel         = "reel"
	AttachmentTypeShare        = "share"
)

// WebhookRequest is the top-level payload for Instagram messaging webhooks.
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
	Delivery  *Delivery       `json:"delivery,omitempty"`
	Read      *Read           `json:"read,omitempty"`
}

// Delivery is a message-delivered receipt.
type Delivery struct {
	Mids      []string `json:"mids,omitempty"`
	Watermark int64    `json:"watermark"`
}

// Read is a message-read receipt.
type Read struct {
	Watermark int64 `json:"watermark"`
}

type Actor struct {
	ID string `json:"id"`
}

type InboundMessage struct {
	Mid         string       `json:"mid"`
	Text        string       `json:"text,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	IsEcho      bool         `json:"is_echo,omitempty"`
	ReplyTo     *ReplyTo     `json:"reply_to,omitempty"`
}

type ReplyTo struct {
	Mid string `json:"mid"`
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

// Postback is sent when a user taps a button.
type Postback struct {
	Mid     string `json:"mid"`
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
// https://developers.instagram.com/docs/instagram-api/reference/user
type UserProfile struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Username   string `json:"username"`
	ProfilePic string `json:"profile_pic"`
}

// VerifyRequest holds parameters for the Instagram webhook verification handshake.
type VerifyRequest struct {
	Mode        string
	Challenge   string
	VerifyToken string
}

// parseVerify extracts verification parameters from webhook query string.
func parseVerify(vals url.Values) *VerifyRequest { //nolint:unused // provider.go will use this
	return &VerifyRequest{
		Mode:        vals.Get("hub.mode"),
		Challenge:   vals.Get("hub.challenge"),
		VerifyToken: vals.Get("hub.verify_token"),
	}
}

// ParseWebhook defensively parses the webhook payload and returns nil if Object is not "instagram".
func (r *WebhookRequest) ParseWebhook() *WebhookRequest {
	if r == nil || r.Object != "instagram" {
		return nil
	}

	return r
}
