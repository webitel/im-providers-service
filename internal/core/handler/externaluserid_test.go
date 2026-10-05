package handler

import (
	"testing"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestParseExternalUserID(t *testing.T) {
	valid := uuid.New()

	got, err := parseExternalUserID(valid.String())
	if err != nil || got != valid {
		t.Fatalf("parseExternalUserID(valid) = %v, %v", got, err)
	}

	for _, in := range []string{"380681130652", "", "not-a-uuid"} {
		_, err := parseExternalUserID(in)
		if status.Code(err) != codes.InvalidArgument {
			t.Errorf("parseExternalUserID(%q) code = %v, want InvalidArgument", in, status.Code(err))
		}
	}
}
