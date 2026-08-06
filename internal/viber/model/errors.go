package model

import "errors"

var (
	ErrTokenInvalid          = errors.New("viber: auth token invalid or revoked")
	ErrReceiverNotSubscribed = errors.New("viber: receiver not subscribed or unreachable")
)
