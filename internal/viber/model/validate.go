package model

import (
	"fmt"
	"strings"
)

// ValidationError is returned when a request is missing one or more required fields.
type ValidationError struct {
	Fields []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("required fields missing: %s", strings.Join(e.Fields, ", "))
}

// requireFields checks that each (fieldName, value) pair is non-empty.
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

// Validate ensures the mandatory fields are present. sender_name is optional here:
// the service derives it from the Viber account info when the caller omits it.
func (r CreateViber) Validate() error {
	return requireFields(
		"name", r.Name,
		"auth_token", r.AuthToken,
		"peer.sub", r.Peer.Sub,
		"peer.iss", r.Peer.Iss,
	)
}
