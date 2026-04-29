package graph

import "net/url"

// VerifyRequest holds params for Facebook webhook verification.
type VerifyRequest struct {
	Mode        string
	Challenge   string
	VerifyToken string
}

// ParseVerify extracts verification parameters from the URL query.
func ParseVerify(vals url.Values) *VerifyRequest {
	return &VerifyRequest{
		Mode:        vals.Get("hub.mode"),
		Challenge:   vals.Get("hub.challenge"),
		VerifyToken: vals.Get("hub.verify_token"),
	}
}
