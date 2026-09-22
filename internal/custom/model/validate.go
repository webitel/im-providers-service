package model

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/webitel/webitel-go-kit/pkg/errors"
)

func requireFields(id string, pairs ...string) error {
	var missing []string

	for i := 0; i+1 < len(pairs); i += 2 {
		if pairs[i+1] == "" {
			missing = append(missing, pairs[i])
		}
	}

	if len(missing) > 0 {
		return errors.InvalidArgument("required fields missing: "+strings.Join(missing, ", "),
			errors.WithID(id))
	}

	return nil
}

func (r CreateCustom) Validate() error {
	const id = "custom.model.create_custom.validate"

	if err := requireFields(id,
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
	const id = "custom.model.update_custom.validate"

	if r.CallbackURL != nil {
		if err := ValidateCallbackURL(*r.CallbackURL); err != nil {
			return err
		}
	}

	if r.AppSecret != nil && strings.TrimSpace(*r.AppSecret) == "" {
		return errors.InvalidArgument("app_secret: must not be empty", errors.WithID(id))
	}

	if r.RequestTimeoutMS != nil {
		if *r.RequestTimeoutMS < MinRequestTimeoutMS || *r.RequestTimeoutMS > MaxRequestTimeoutMS {
			return errors.InvalidArgument(
				fmt.Sprintf("request_timeout_ms: must be between %d and %d", MinRequestTimeoutMS, MaxRequestTimeoutMS),
				errors.WithID(id),
			)
		}
	}

	if r.RetryAttempts != nil {
		if *r.RetryAttempts < 0 || *r.RetryAttempts > MaxRetryAttempts {
			return errors.InvalidArgument(
				fmt.Sprintf("retry_attempts: must be between 0 and %d", MaxRetryAttempts),
				errors.WithID(id),
			)
		}
	}

	if r.AllowedIPs != nil {
		return ValidateAllowedIPs(*r.AllowedIPs)
	}

	return nil
}

func ValidateCallbackURL(raw string) error {
	const id = "custom.model.validate_callback_url"

	parsed, err := url.Parse(raw)
	if err != nil {
		return errors.InvalidArgument("callback_url: not a valid url", errors.WithCause(err), errors.WithID(id))
	}

	if parsed.Host == "" {
		return errors.InvalidArgument("callback_url: must be absolute", errors.WithID(id))
	}

	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return nil
	default:
		return errors.InvalidArgument("callback_url: scheme must be http or https", errors.WithID(id))
	}
}
