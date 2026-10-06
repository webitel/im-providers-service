package model

import "errors"

var (
	ErrKeyInvalid              = errors.New("viber_bm: infobip api key invalid or revoked")
	ErrReceiverNotReachable    = errors.New("viber_bm: receiver not reachable or opted out")
	ErrMediaMissing            = errors.New("viber_bm: media url missing")
	ErrFileTypeUnsupported     = errors.New("viber_bm: file type not supported by viber business messages")
	ErrAudioUnsupported        = errors.New("viber_bm: audio is not supported by viber business messages")
	ErrVideoTooLarge           = errors.New("viber_bm: video exceeds 200 MB")
	ErrVideoTooLong            = errors.New("viber_bm: video exceeds 600 seconds")
	ErrVideoPreviewUnavailable = errors.New("viber_bm: storage has no thumbnail or duration for this video")
	ErrSessionWindowRequired   = errors.New("viber_bm: free-form message requires an open 24h session")
)
