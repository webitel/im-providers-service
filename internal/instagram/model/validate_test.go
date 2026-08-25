package model

import (
	"errors"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// CreateInstagram.Validate
// ---------------------------------------------------------------------------

func TestValidate_AllFieldsMissing(t *testing.T) {
	err := CreateInstagram{}.Validate()
	if err == nil {
		t.Fatal("expected validation error for empty struct")
	}

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T", err)
	}

	for _, field := range []string{"name", "meta_app_id", "business_account_id", "ig_access_token"} {
		found := false

		for _, f := range ve.Fields {
			if f == field {
				found = true

				break
			}
		}

		if !found {
			t.Errorf("expected field %q in validation error, got %v", field, ve.Fields)
		}
	}
}

func TestValidate_PartialMissing(t *testing.T) {
	err := CreateInstagram{
		Name:    "My IG Gate",
		IGToken: "token",
	}.Validate()
	if err == nil {
		t.Fatal("expected validation error when meta_app_id and business_account_id are missing")
	}

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T", err)
	}

	if len(ve.Fields) != 2 {
		t.Errorf("expected 2 missing fields, got %v", ve.Fields)
	}
}

func TestValidate_AllPresent_NoError(t *testing.T) {
	err := CreateInstagram{
		Name:              "IG Gate",
		MetaAppID:         "app-1",
		BusinessAccountID: "biz-1",
		IGToken:           "token",
	}.Validate()
	if err != nil {
		t.Errorf("expected no error for fully populated request, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// ValidationError.Error
// ---------------------------------------------------------------------------

func TestValidationError_ErrorString(t *testing.T) {
	ve := &ValidationError{Fields: []string{"name", "meta_app_id"}}
	msg := ve.Error()

	if !strings.Contains(msg, "name") || !strings.Contains(msg, "meta_app_id") {
		t.Errorf("error string must list missing fields, got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// UpdateInstagram.ApplyTo
// ---------------------------------------------------------------------------

func TestApplyTo_NilPointers_Noop(t *testing.T) {
	gate := &InstagramGate{Name: "Original", Enabled: false}

	UpdateInstagram{ID: "g-1"}.ApplyTo(gate)

	if gate.Name != "Original" {
		t.Errorf("name must not change when update.Name is nil")
	}
}

func TestApplyTo_AllFields_Applied(t *testing.T) {
	gate := &InstagramGate{Name: "Old", Enabled: false, IGToken: "old-tok"}

	newName := "New"
	newToken := "new-tok"
	enabled := true

	UpdateInstagram{
		ID:      "g-1",
		Name:    &newName,
		IGToken: &newToken,
		Enabled: &enabled,
	}.ApplyTo(gate)

	if gate.Name != "New" {
		t.Errorf("expected name=New, got %s", gate.Name)
	}

	if gate.IGToken != "new-tok" {
		t.Errorf("expected token=new-tok, got %s", gate.IGToken)
	}

	if !gate.Enabled {
		t.Error("expected enabled=true")
	}
}
