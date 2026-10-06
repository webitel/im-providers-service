package model

import (
	"errors"
	"testing"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

func TestValidateBaseURL(t *testing.T) {
	cases := []struct {
		url  string
		want bool
	}{
		{"https://xyz.api.infobip.com", true},
		{"https://xyz.api.infobip.com/", true},
		{"http://xyz.api.infobip.com", false},
		{"xyz.api.infobip.com", false},
		{"https://", false},
		{"https://user:pass@xyz.api.infobip.com", false},
		{"ftp://xyz.api.infobip.com", false},
		{"://bad", false},
	}

	for _, tc := range cases {
		err := validateBaseURL(tc.url)
		if (err == nil) != tc.want {
			t.Errorf("validateBaseURL(%q) = %v, want ok=%v", tc.url, err, tc.want)
		}

		var ve *ValidationError
		if err != nil && !errors.As(err, &ve) {
			t.Errorf("validateBaseURL(%q) error is not a ValidationError", tc.url)
		}
	}
}

func TestCreateValidateChecksBaseURL(t *testing.T) {
	req := CreateViberBM{
		Name: "n", BaseURL: "http://10.0.0.1", SenderName: "s", APIKey: "k",
		Peer: sharedmodel.Peer{Sub: "a", Iss: "b"},
	}

	if err := req.Validate(); err == nil {
		t.Fatal("expected http base_url to be rejected")
	}

	req.BaseURL = "https://xyz.api.infobip.com"
	if err := req.Validate(); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}
}

func TestUpdateValidateSkipsUnchangedBaseURL(t *testing.T) {
	empty := ""
	if err := (UpdateViberBM{BaseURL: &empty}).Validate(); err != nil {
		t.Fatalf("empty base_url means keep existing, got %v", err)
	}

	if err := (UpdateViberBM{}).Validate(); err != nil {
		t.Fatalf("nil base_url means keep existing, got %v", err)
	}
}
