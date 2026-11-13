package graphapi

import (
	"testing"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

func TestSendErrorCode(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{"user unavailable 551", 400, `{"error":{"message":"(#551) This person isn't available right now.","code":551}}`, sharedmodel.SendErrRecipientUnavailable},
		{"user unavailable subcode", 400, `{"error":{"message":"Message Not Sent","code":200,"error_subcode":1545041}}`, sharedmodel.SendErrRecipientUnavailable},
		{"outside window", 400, `{"error":{"message":"(#10) This message is sent outside of allowed window.","code":10,"error_subcode":2018278}}`, sharedmodel.SendErrRecipientUnavailable},
		{"no matching user", 400, `{"error":{"message":"(#100) No matching user found","code":100,"error_subcode":2018001}}`, sharedmodel.SendErrRecipientUnavailable},
		{"fetch failed subcode", 400, `{"error":{"message":"(#100) Failed to fetch the file from the url","code":100,"error_subcode":2018008}}`, sharedmodel.SendErrFileUnavailable},
		{"upload failure text", 400, `{"error":{"message":"(#100) Upload attachment failure.","code":100}}`, sharedmodel.SendErrFileUnavailable},
		{"size exceeded", 400, `{"error":{"message":"(#100) Attachment size exceeds allowable limit","code":100}}`, sharedmodel.SendErrFileTooLarge},
		{"unsupported type", 400, `{"error":{"message":"(#100) This attachment type is not supported","code":100}}`, sharedmodel.SendErrUnsupportedMedia},
		{"rate limited", 400, `{"error":{"message":"(#613) Calls to this api have exceeded the rate limit.","code":613}}`, sharedmodel.SendErrChannelUnavailable},
		{"page token expired", 400, `{"error":{"message":"Error validating access token: Session has expired.","code":190}}`, sharedmodel.SendErrChannelUnavailable},
		{"temporary failure", 500, `{"error":{"message":"(#1200) Temporary send message failure","code":1200}}`, sharedmodel.SendErrChannelUnavailable},
		{"bad param", 400, `{"error":{"message":"(#100) Param recipient[id] must be a valid ID string","code":100}}`, ""},
		{"5xx non-json", 502, `<html>bad gateway</html>`, sharedmodel.SendErrChannelUnavailable},
		{"429 empty", 429, ``, sharedmodel.SendErrChannelUnavailable},
		{"unknown error code 1", 400, `{"error":{"message":"(#1) An unknown error occurred","code":1}}`, sharedmodel.SendErrChannelUnavailable},
		{"service unavailable", 400, `{"error":{"message":"(#2) Service temporarily unavailable","code":2}}`, sharedmodel.SendErrChannelUnavailable},
		{"app rate limit", 400, `{"error":{"message":"(#4) Application request limit reached","code":4}}`, sharedmodel.SendErrChannelUnavailable},
		{"user rate limit", 400, `{"error":{"message":"(#17) User request limit reached","code":17}}`, sharedmodel.SendErrChannelUnavailable},
		{"page rate limit", 400, `{"error":{"message":"(#32) Page request limit reached","code":32}}`, sharedmodel.SendErrChannelUnavailable},
		{"messenger rate limit", 400, `{"error":{"message":"(#80006) There have been too many messenger api calls","code":80006}}`, sharedmodel.SendErrChannelUnavailable},
		{"unmapped 4xx", 400, `{"error":{"message":"(#200) Permissions error","code":200}}`, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SendErrorCode(tc.status, []byte(tc.body)); got != tc.want {
				t.Errorf("SendErrorCode(%d, %s) = %q, want %q", tc.status, tc.body, got, tc.want)
			}
		})
	}
}

func TestParseError(t *testing.T) {
	e, ok := ParseError([]byte(`{"error":{"message":"boom","code":100,"error_subcode":2018001}}`))
	if !ok || e.Code != 100 || e.ErrorSubcode != 2018001 || e.Message != "boom" {
		t.Errorf("ParseError = %+v, %v", e, ok)
	}

	if _, ok := ParseError([]byte(`{"message_id":"m.1"}`)); ok {
		t.Error("a success body must not parse as an error")
	}
}
