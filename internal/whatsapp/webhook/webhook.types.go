package webhook

type WhatsappApiNotificationPayloadSchemaType struct {
	Object string  `json:"object"`
	Entry  []Entry `json:"entry"`
}

type Entry struct {
	ID      string   `json:"id"`
	Time    int64    `json:"time,omitempty"`
	Changes []Change `json:"changes"`
}

type WebhookFieldEnum string

const (
	WebhookFieldEnumMessages WebhookFieldEnum = "messages"
)

type Change struct {
	Value any              `json:"value"`
	Field WebhookFieldEnum `json:"field"`
}

type MessagesValue struct {
	MessagingProduct string          `json:"messaging_product"`
	Metadata         Metadata        `json:"metadata"`
	Contacts         []SenderContact `json:"contacts,omitempty"`
	Statuses         []Status        `json:"statuses,omitempty"`
	Errors           []Error         `json:"errors,omitempty"`
	Messages         []Message       `json:"messages,omitempty"`
}

type Error struct {
	Code      int    `json:"code"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	Href      string `json:"href"`
	ErrorData struct {
		Details string `json:"details"`
	} `json:"error_data"`
}

type Metadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type SenderContact struct {
	WaID            string
	IdentityHashKey string
	Profile         Profile
}

type Profile struct {
	Name string `json:"name"`
}

type Status struct {
	ID                       string       `json:"id"`
	Conversation             Conversation `json:"conversation"`
	Errors                   []Error      `json:"errors"`
	Status                   string       `json:"status"`
	Timestamp                string       `json:"timestamp"`
	RecipientID              string       `json:"recipient_id"`
	RecipientType            string       `json:"recipient_type,omitempty"`              // Only included if message sent to a group
	RecipientParticipantId   string       `json:"recipient_participant_id,omitempty"`    // Only included if message sent to a group
	RecipientIdentityKeyHash string       `json:"recipient_identity_key_hash,omitempty"` // Only included if identity change check enabled
	BizOpaqueCallbackData    string       `json:"biz_opaque_callback_data,omitempty"`    // Only included if message sent with biz_opaque_callback_data
	Pricing                  Pricing      `json:"pricing,omitempty"`
}

type Pricing struct {
	Billable     bool                      `json:"billable"`
	PricingModel string                    `json:"pricing_model"`
	Category     MessageStatusCategoryEnum `json:"category"`
}

type Conversation struct {
	ID                  string `json:"id"`
	ExpirationTimestamp string `json:"expiration_timestamp,omitempty"`
	Origin              Origin `json:"origin,omitempty"`
}

type Origin struct {
	Type                MessageStatusCategoryEnum `json:"type"`
	ExpirationTimestamp string                    `json:"expiration_timestamp,omitempty"`
}

type MessageStatusCategoryEnum string

const (
	MessageStatusCategoryEnumSent MessageStatusCategoryEnum = "sent"
)

type Message struct {
	Id                                           string                                      `json:"id"`
	From                                         string                                      `json:"from"`
	Timestamp                                    string                                      `json:"timestamp"`
	Type                                         NotificationMessageTypeEnum                 `json:"type"`
	GroupId                                      string                                      `json:"group_id,omitempty"`
	Context                                      NotificationPayloadMessageContextSchemaType `json:"context,omitempty"`
	Errors                                       []Error                                     `json:"errors,omitempty"`
	NotificationPayloadTextMessageSchemaType     `json:",inline"`
	NotificationPayloadImageMessageSchemaType    `json:",inline"`
	NotificationPayloadDocumentMessageSchemaType `json:",inline"`
}

type NotificationMessageTypeEnum string

const (
	NotificationMessageTypeText        NotificationMessageTypeEnum = "text"
	NotificationMessageTypeAudio       NotificationMessageTypeEnum = "audio"
	NotificationMessageTypeImage       NotificationMessageTypeEnum = "image"
	NotificationMessageTypeButton      NotificationMessageTypeEnum = "button"
	NotificationMessageTypeDocument    NotificationMessageTypeEnum = "document"
	NotificationMessageTypeOrder       NotificationMessageTypeEnum = "order"
	NotificationMessageTypeSticker     NotificationMessageTypeEnum = "sticker"
	NotificationMessageTypeSystem      NotificationMessageTypeEnum = "system"
	NotificationMessageTypeVideo       NotificationMessageTypeEnum = "video"
	NotificationMessageTypeReaction    NotificationMessageTypeEnum = "reaction"
	NotificationMessageTypeInteractive NotificationMessageTypeEnum = "interactive"
	NotificationMessageTypeUnknown     NotificationMessageTypeEnum = "unknown"
	NotificationMessageTypeLocation    NotificationMessageTypeEnum = "location"
	NotificationMessageTypeContacts    NotificationMessageTypeEnum = "contacts"
	NotificationMessageTypeUnsupported NotificationMessageTypeEnum = "unsupported"
)

type NotificationPayloadMessageContextSchemaType struct {
	Forwarded           bool   `json:"forwarded,omitempty"`
	FrequentlyForwarded bool   `json:"frequently_forwarded,omitempty"`
	From                string `json:"from,omitempty"`
	Id                  string `json:"id"`
	ReferredProduct     struct {
		CatalogId         string `json:"catalog_id"`
		ProductRetailerId string `json:"product_retailer_id"`
	} `json:"referred_product,omitempty"`
}

type NotificationPayloadTextMessageSchemaType struct {
	Text struct {
		Body string `json:"body"`
	} `json:"text,omitempty"`
}

type NotificationPayloadImageMessageSchemaType struct {
	Image struct {
		Id       string `json:"id"`
		MIMEType string `json:"mime_type"`
		SHA256   string `json:"sha256"`
		Caption  string `json:"caption,omitempty"`
		Url      string `json:"url,omitempty"`
	} `json:"image,omitempty"`
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
