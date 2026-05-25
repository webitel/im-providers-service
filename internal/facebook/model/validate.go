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
// Call it as requireFields("name", r.Name, "page_id", r.PageID, ...).
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

func (r CreateFacebook) Validate() error {
	return requireFields(
		"name", r.Name,
		"meta_app_id", r.MetaAppID,
		"page_id", r.PageID,
		"page_token", r.PageToken,
	)
}

func (r CreateMetaApp) Validate() error {
	return requireFields(
		"name", r.Name,
		"uri", r.URI,
		"app_id", r.AppID,
		"app_secret", r.AppSecret,
		"oauth_redirect_uri", r.OAuthRedirectURI,
		"verify_token", r.VerifyToken,
	)
}
