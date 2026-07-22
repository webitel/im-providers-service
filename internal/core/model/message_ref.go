package model

import (
	"time"

	"github.com/google/uuid"
)

// MessageRef maps a provider-assigned message id back to the internal
// message context, so webhook receipts (delivered/read/failed) can be
// reported to im-thread-service per recipient.
type MessageRef struct {
	GateID            string
	ProviderMessageID string
	// ProviderUserID is the recipient platform-specific id (PSID / wa_id)
	// used by watermark-based receipts that carry no message id.
	ProviderUserID string
	MessageID      uuid.UUID
	ThreadID       uuid.UUID
	// MemberID is the recipient contact id (thread member).
	MemberID uuid.UUID
	DomainID int64
	SentAt   time.Time
}
