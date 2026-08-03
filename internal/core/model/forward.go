package model

type ForwardOriginKind int32

const (
	ForwardOriginUnspecified ForwardOriginKind = iota
	ForwardOriginInternal
	ForwardOriginExternalUser
	ForwardOriginExternalHiddenUser
	ForwardOriginExternalChat
)

type ForwardOrigin struct {
	Kind           ForwardOriginKind `json:"kind"`
	SenderName     string            `json:"sender_name,omitempty"`
	OriginalSentAt int64             `json:"original_sent_at,omitempty"`
}
