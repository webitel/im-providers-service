package model

import (
	"fmt"
	"net/url"
	"strings"
)

type ValidationError struct {
	Fields []string
	Reason string
}

func (e *ValidationError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("invalid %s: %s", strings.Join(e.Fields, ", "), e.Reason)
	}

	return fmt.Sprintf("required fields missing: %s", strings.Join(e.Fields, ", "))
}

func requireFields(pairs ...string) error {
	var missing []string

	for i := 0; i+1 < len(pairs); i += 2 {
		if pairs[i+1] == "" {
			missing = append(missing, pairs[i])
		}
	}

	if len(missing) > 0 {
		return &ValidationError{Fields: missing}
	}

	return nil
}

func (r CreateViberBM) Validate() error {
	if err := requireFields(
		"name", r.Name,
		"base_url", r.BaseURL,
		"sender_name", r.SenderName,
		"api_key", r.APIKey,
		"peer.sub", r.Peer.Sub,
		"peer.iss", r.Peer.Iss,
	); err != nil {
		return err
	}

	return validateBaseURL(r.BaseURL)
}

func (r UpdateViberBM) Validate() error {
	if r.BaseURL == nil || *r.BaseURL == "" {
		return nil
	}

	return validateBaseURL(*r.BaseURL)
}

// validateBaseURL accepts only an https origin: the gate's API key is sent to
// this host, so anything else would leak it or reach internal services.
func validateBaseURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return &ValidationError{Fields: []string{"base_url"}, Reason: "must be an https url"}
	}

	return nil
}
