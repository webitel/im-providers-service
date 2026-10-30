package bot

import (
	"testing"

	coremodel "github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/im-providers-service/internal/telegram/bot/model"
)

func ptr(s string) *string { return &s }

func TestToExternalUser(t *testing.T) {
	p := &Provider{}

	got := p.toExternalUser(&model.User{ID: 42, FirstName: "Test", LastName: ptr("User"), Username: ptr("test_handle")})
	if got.ID != "42" || got.FirstName != "Test" || got.LastName != "User" || got.Username != "test_handle" {
		t.Errorf("toExternalUser = %+v", got)
	}

	got = p.toExternalUser(&model.User{ID: 42, FirstName: "Test"})
	if got.LastName != "" || got.Username != "" {
		t.Errorf("optional fields must stay empty, got %+v", got)
	}
}

func TestNewContactRequest(t *testing.T) {
	tests := []struct {
		name         string
		user         *coremodel.ExternalUser
		wantName     string
		wantUsername string
	}{
		{"full profile", &coremodel.ExternalUser{ID: "42", FirstName: "Test", LastName: "User", Username: "test_handle"}, "Test User", "test_handle"},
		{"no last name", &coremodel.ExternalUser{ID: "42", FirstName: "Test", Username: "test_handle"}, "Test", "test_handle"},
		// The contact username must not be empty: no handle falls back to the user id.
		{"no handle, no last name", &coremodel.ExternalUser{ID: "42", FirstName: "Test"}, "Test", "42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newContactRequest("telegram_bot", tt.user)

			if req.GetName() != tt.wantName {
				t.Errorf("Name = %q, want %q", req.GetName(), tt.wantName)
			}

			if req.GetUsername() != tt.wantUsername {
				t.Errorf("Username = %q, want %q", req.GetUsername(), tt.wantUsername)
			}

			if req.GetSubject() != "42" || req.GetIssId() != "telegram_bot" || req.GetType() != "telegram_bot" {
				t.Errorf("Subject/IssId/Type = %q/%q/%q", req.GetSubject(), req.GetIssId(), req.GetType())
			}
		})
	}
}

func TestExternalUserHash_Username(t *testing.T) {
	a := coremodel.ExternalUser{ID: "42", FirstName: "Test"}
	b := a
	b.Username = "test_handle"

	if a.Hash() == b.Hash() {
		t.Error("a username change must change the hash")
	}
}
