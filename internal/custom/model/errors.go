package model

import "errors"

var (
	ErrSignatureInvalid = errors.New("custom: request signature invalid")
	ErrSourceNotAllowed = errors.New("custom: source address not allowed")
	ErrCallbackRejected = errors.New("custom: external system rejected the payload")
	ErrCallbackFailed   = errors.New("custom: external system unreachable")
	ErrChatUnknown      = errors.New("custom: no chat known for this recipient")
)
