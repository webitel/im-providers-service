package model

import "errors"

var (
	ErrKeyInvalid            = errors.New("viber_bm: infobip api key invalid or revoked")
	ErrReceiverNotReachable  = errors.New("viber_bm: receiver not reachable or opted out")
	ErrMediaMissing          = errors.New("viber_bm: media url missing")
	ErrSessionWindowRequired = errors.New("viber_bm: free-form message requires an open 24h session")
)
