package model

type DeliveryStatus int

const (
	DeliveryUnspecified DeliveryStatus = iota
	DeliveryDelivered
	DeliveryRead
	DeliveryFailed
)

func (s DeliveryStatus) String() string {
	switch s {
	case DeliveryDelivered:
		return "delivered"
	case DeliveryRead:
		return "read"
	case DeliveryFailed:
		return "failed"
	default:
		return "unspecified"
	}
}

type MessageDeliveryReport struct {
	GateID     string
	ExternalID string
	Status     DeliveryStatus
	Reason     string
	At         int64

	// DomainID and Sub identify the external user the report is about. The gateway
	// authenticates provider calls by resolving a contact from "{DomainID}.{Sub}".
	DomainID int64
	Sub      string
}
