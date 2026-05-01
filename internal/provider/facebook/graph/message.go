package graph

type OutboundPayload struct {
	Type      string    `json:"messaging_type"`
	Recipient Recipient `json:"recipient"`
	Message   Message   `json:"message"`
}

type Recipient struct {
	ID string `json:"id"`
}

type Message struct {
	Text       string      `json:"text,omitempty"`
	Attachment *Attachment `json:"attachment,omitempty"`
}

type Attachment struct {
	Type    string `json:"type"`
	Payload URL    `json:"payload"`
}

type URL struct {
	URL string `json:"url"`
}

func NewTextRequest(psid, text string) OutboundPayload {
	return OutboundPayload{
		Type:      "RESPONSE",
		Recipient: Recipient{ID: psid},
		Message:   Message{Text: text},
	}
}

func NewMediaRequest(psid, mType, url string) OutboundPayload {
	return OutboundPayload{
		Type:      "RESPONSE",
		Recipient: Recipient{ID: psid},
		Message: Message{
			Attachment: &Attachment{
				Type:    mType,
				Payload: URL{URL: url},
			},
		},
	}
}
