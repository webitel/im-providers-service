package messaging

type Message struct {
	ID                                           string                      `json:"id"`
	From                                         string                      `json:"from"`
	Timestamp                                    string                      `json:"string"`
	Type                                         NotificationMessageTypeEnum `json:"type"`
	Errors                                       []Error                     `json:"errors,omitempty"`
	NotificationPayloadTextMessageSchemaType     `json:",inline"`
	NotificationPayloadImageMessageSchemaType    `json:",inline"`
	NotificationPayloadDocumentMessageSchemaType `json:",inline"`
}

type NotificationPayloadTextMessageSchemaType struct {
	Text struct {
		Body string `json:"body"`
	} `json:"text"`
}

type NotificationPayloadImageMessageSchemaType struct {
	Image struct {
		ID       string `json:"id"`
		MIMEType string `json:"mime_type"`
		SHA256   string `json:"sha_256"`
		Caption  string `json:"caption"`
		URL      string `json:"url"`
	} `json:"image"`
}

type NotificationPayloadDocumentMessageSchemaType struct {
	Document struct {
		Id       string `json:"id"`
		MIMEType string `json:"mime_type"`
		SHA256   string `json:"sha256"`
		Caption  string `json:"caption,omitempty"`
		Filename string `json:"filename,omitempty"`
		Link     string `json:"link,omitempty"`
	} `json:"document,omitempty"`
}
