package model

import (
	"fmt"
	"strings"
)

type ValidationError struct {
	Fields []string
}

func (e *ValidationError) Error() string {
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
	return requireFields(
		"name", r.Name,
		"base_url", r.BaseURL,
		"sender_name", r.SenderName,
		"api_key", r.APIKey,
		"peer.sub", r.Peer.Sub,
		"peer.iss", r.Peer.Iss,
	)
}
