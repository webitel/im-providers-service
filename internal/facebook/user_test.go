package facebook

import (
	"testing"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

func TestNewContactRequest(t *testing.T) {
	tests := []struct {
		name     string
		user     *sharedmodel.ExternalUser
		wantName string
	}{
		{"first and last name", &sharedmodel.ExternalUser{ID: "psid-1", FirstName: "Taras", LastName: "Shevchenko"}, "Taras Shevchenko"},
		{"first name only", &sharedmodel.ExternalUser{ID: "psid-1", FirstName: "Taras"}, "Taras"},
		{"no name", &sharedmodel.ExternalUser{ID: "psid-1"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newContactRequest("facebook", tt.user)

			if req.GetName() != tt.wantName {
				t.Errorf("Name = %q, want %q", req.GetName(), tt.wantName)
			}
			// Username is the PSID: unique per issuer, never the last name.
			if req.GetUsername() != "psid-1" || req.GetSubject() != "psid-1" {
				t.Errorf("Username/Subject = %q/%q, want psid-1", req.GetUsername(), req.GetSubject())
			}

			if req.GetIssId() != "facebook" || req.GetType() != "facebook" {
				t.Errorf("IssId/Type = %q/%q, want facebook", req.GetIssId(), req.GetType())
			}
		})
	}
}
