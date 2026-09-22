package model

import "github.com/webitel/webitel-go-kit/pkg/errors"

var (
	ErrSignatureInvalid = errors.Unauthenticated("custom: request signature invalid")
	ErrSourceNotAllowed = errors.Forbidden("custom: source address not allowed")
	ErrCallbackRejected = errors.Aborted("custom: external system rejected the payload")
	ErrCallbackFailed   = errors.Unavailable("custom: external system unreachable")
	ErrChatUnknown      = errors.NotFound("custom: no chat known for this recipient")
)
