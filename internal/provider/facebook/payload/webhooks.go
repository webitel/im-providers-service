package payload

// WebhookRequest is the top-level Meta webhook structure.
type WebhookRequest struct {
	Object string `json:"object"`
	Entry  []struct {
		ID        string      `json:"id"`
		Time      int64       `json:"time"`
		Messaging []Messaging `json:"messaging"`
	} `json:"entry"`
}

type Messaging struct {
	Sender    struct{ ID string } `json:"sender"`
	Recipient struct{ ID string } `json:"recipient"`
	Timestamp int64               `json:"timestamp"`
	Message   *InboundMessage     `json:"message,omitempty"`
	Postback  *InboundPostback    `json:"postback,omitempty"`
}

type InboundMessage struct {
	Mid         string              `json:"mid"`
	Text        string              `json:"text,omitempty"`
	Attachments []InboundAttachment `json:"attachments,omitempty"`
}

type InboundAttachment struct {
	Type    string `json:"type"`
	Payload struct {
		URL   string `json:"url,omitempty"`
		Title string `json:"title,omitempty"`
		Name  string `json:"name,omitempty"`
	} `json:"payload"`
}
type InboundPostback struct {
	Title   string `json:"title"`
	Payload string `json:"payload"` // The data sent back from a button click
}

// AllMessages flattens the entries into a single slice for iteration.
func (r *WebhookRequest) AllMessages() []Messaging {
	var res []Messaging
	for _, entry := range r.Entry {
		res = append(res, entry.Messaging...)
	}
	return res
}
