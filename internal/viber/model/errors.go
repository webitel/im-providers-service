package model

import "errors"

var (
	// ErrTokenInvalid is returned when Viber rejects the bot auth token (status 2).
	ErrTokenInvalid = errors.New("viber: auth token invalid or revoked")
	// ErrReceiverNotSubscribed is returned when the recipient cannot be reached
	// because they are not registered, not subscribed, or have blocked the bot
	// (status 5/6/7). Callers should surface this as a delivery precondition failure.
	ErrReceiverNotSubscribed = errors.New("viber: receiver not subscribed or unreachable")
)
