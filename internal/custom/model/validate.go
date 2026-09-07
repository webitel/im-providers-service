package model

import (
	"fmt"
	"net/url"
	"strings"
)

type ValidationError struct {
	Fields []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("required fields missing: %s", strings.Join(e.Fields, ", "))
}

type FieldError struct {
	Field  string
	Reason string
}

func (e *FieldError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
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

func (r CreateCustom) Validate() error {
	if err := requireFields(
		"name", r.Name,
		"callback_url", r.CallbackURL,
		"peer.sub", r.Peer.Sub,
		"peer.iss", r.Peer.Iss,
	); err != nil {
		return err
	}

	if err := ValidateCallbackURL(r.CallbackURL); err != nil {
		return err
	}

	return ValidateAllowedIPs(r.AllowedIPs)
}

func (r UpdateCustom) Validate() error {
	if r.CallbackURL != nil && *r.CallbackURL != "" {
		if err := ValidateCallbackURL(*r.CallbackURL); err != nil {
			return err
		}
	}

	if r.AllowedIPs != nil {
		return ValidateAllowedIPs(*r.AllowedIPs)
	}

	return nil
}

func ValidateCallbackURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return &FieldError{Field: "callback_url", Reason: "not a valid url"}
	}

	if parsed.Host == "" {
		return &FieldError{Field: "callback_url", Reason: "must be absolute"}
	}

	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return nil
	default:
		return &FieldError{Field: "callback_url", Reason: "scheme must be http or https"}
	}
}
