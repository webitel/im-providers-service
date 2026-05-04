package components

import (
	"cmp"
	"encoding/json"

	"github.com/webitel/webitel-go-kit/pkg/errors"
)

type AddressType string

const (
	HomeAddress AddressType = "HOME"
	WorkAddress AddressType = "WORK"
)

type URLType string

const (
	HomeURL URLType = "HOME"
	WorkURL URLType = "WORK"
)

type EmailType string

const (
	HomeEmail EmailType = "HOME"
	WorkEmail EmailType = "WORK"
)

type PhoneType string

const (
	CellPhone   PhoneType = "CELL"
	MainPhone   PhoneType = "MAIN"
	IphonePhone PhoneType = "IPHONE"
	HomePhone   PhoneType = "HOME"
	WorkPhone   PhoneType = "WORK"
)

type ContactAddress struct {
	Street      string      `json:"street,omitempty"`
	City        string      `json:"city,omitempty"`
	State       string      `json:"state,omitempty"`
	Zip         string      `json:"zip,omitempty"`
	Country     string      `json:"country,omitempty"`
	CountryCode string      `json:"countryCode,omitempty"`
	Type        AddressType `json:"type"`
}

type ContactName struct {
	FormattedName string `json:"formatted_name"`
	FirstName     string `json:"first_name,omitempty"`
	LastName      string `json:"last_name,omitempty"`
	MiddleName    string `json:"middle_name,omitempty"`
	Suffix        string `json:"suffix,omitempty"`
	Prefix        string `json:"prefix,omitempty"`
}

type ContactOrg struct {
	Company    string `json:"company,omitempty"`
	Title      string `json:"title,omitempty"`
	Department string `json:"department,omitempty"`
}

type ContactEmail struct {
	Email string    `json:"email,omitempty"`
	Type  EmailType `json:"type,omitempty"`
}

type ContactPhone struct {
	Phone string    `json:"phone,omitempty"`
	WaId  string    `json:"wa_id,omitempty"`
	Type  PhoneType `json:"type"`
}

type ContactUrl struct {
	Url  string  `json:"url"`
	Type URLType `json:"type"`
}

type Contact struct {
	Name      ContactName      `json:"name"`
	Org       ContactOrg       `json:"org"`
	Addresses []ContactAddress `json:"addresses,omitempty"`
	Urls      []ContactUrl     `json:"urls,omitempty"`
	Emails    []ContactEmail   `json:"emails,omitempty"`
	Phones    []ContactPhone   `json:"phones,omitempty"`
	Birthday  string           `json:"birthday,omitempty"`
}

func (contact *Contact) AsMetadata() map[string]any {
	if contact == nil {
		return nil
	}

	metadata := make(map[string]any)

	if contact.Birthday != "" {
		metadata["birthday"] = contact.Birthday
	}

	if cmp.Or(contact.Org.Company, contact.Org.Department, contact.Org.Title) != "" {
		metadata["company"] = map[string]string{
			"company":    contact.Org.Company,
			"department": contact.Org.Department,
			"title":      contact.Org.Title,
		}
	}

	metadata["name_details"] = map[string]any{
		"first_name":  contact.Name.FirstName,
		"last_name":   contact.Name.LastName,
		"middle_name": contact.Name.MiddleName,
		"prefix":      contact.Name.Prefix,
		"suffix":      contact.Name.Suffix,
	}

	if len(contact.Addresses) > 0 {
		metadata["addresses"] = contact.Addresses
	}

	if len(contact.Emails) > 1 {
		metadata["emails"] = contact.Emails[1:]
	}

	if len(contact.Urls) > 0 {
		metadata["urls"] = contact.Urls
	}

	if len(contact.Phones) > 1 {
		metadata["phones"] = contact.Phones[1:]
	}

	return metadata
}

func NewContact(name ContactName) *Contact {
	return &Contact{
		Name: name,
	}
}

type ContactMessage struct {
	Contacts []Contact `json:"contacts" validate:"required"`
}

type ContactMessageConfigs struct {
	Name string `json:"name" validate:"required"`
}

type ContactMessageApiPayload struct {
	BaseMessagePayload
	Contacts []Contact `json:"contacts" validate:"required"`
}

func NewContactMessage(configs []Contact) *ContactMessage {
	return &ContactMessage{
		Contacts: configs,
	}
}

func (m *ContactMessage) ToJson(configs ApiCompatibleJsonConverterConfigs) ([]byte, error) {
	jsonData := ContactMessageApiPayload{
		BaseMessagePayload: CreateBaseMessagePayload(configs.SendToPhoneNumber(), MessageTypeContact),
		Contacts:           m.Contacts,
	}

	if configs.ReplyToMessageID() != "" {
		jsonData.MessageContext = &MessageContext{
			MessageID: configs.ReplyToMessageID(),
		}
	}
	jsonToReturn, err := json.Marshal(jsonData)

	if err != nil {
		return nil, errors.Internal("marshaling json", errors.WithCause(err), errors.WithID("whatsapp.messaging.components.contact.to_json"))
	}
	return jsonToReturn, nil
}
