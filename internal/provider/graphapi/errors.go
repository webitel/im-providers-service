// Package graphapi holds Meta Graph API helpers shared by the Messenger and
// Instagram channels.
package graphapi

import (
	"encoding/json"
	"strings"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

// Error is the Graph API error envelope body.
// https://developers.facebook.com/docs/graph-api/guides/error-handling
type Error struct {
	Message      string `json:"message"`
	Type         string `json:"type"`
	Code         int    `json:"code"`
	ErrorSubcode int    `json:"error_subcode"`
	ErrorUserMsg string `json:"error_user_msg"`
}

// ParseError extracts the Graph error from a response body; ok is false when
// the body is not a Graph error envelope.
func ParseError(body []byte) (Error, bool) {
	var envelope struct {
		Error *Error `json:"error"`
	}

	if err := json.Unmarshal(body, &envelope); err != nil || envelope.Error == nil {
		return Error{}, false
	}

	return *envelope.Error, true
}

// Messenger Platform send error codes/subcodes.
// https://developers.facebook.com/docs/messenger-platform/error-codes
const (
	codeUnknown            = 1
	codeServiceUnavailable = 2
	codeTooManyCalls       = 4
	codeUserRequestLimit   = 17
	codePageRequestLimit   = 32
	codeAccessToken        = 190
	codeRateLimit          = 613
	codeTemporaryFailure   = 1200
	codeMessengerRateLimit = 80006
	codeUserUnavailable    = 551

	subcodeUserUnavailable = 1545041
	subcodeNoMatchingUser  = 2018001
	subcodeOutsideWindow   = 2018278
	subcodeOutsideWindowV1 = 2018065
	subcodeFetchFailed     = 2018008
	subcodeUploadFailed    = 2018047
)

// SendErrorCode maps a failed Graph send to a normalized code, "" when
// neither the error body nor the HTTP status say anything specific.
func SendErrorCode(statusCode int, body []byte) string {
	if e, ok := ParseError(body); ok {
		if code := e.sendErrorCode(); code != "" {
			return code
		}
	}

	return sharedmodel.SendErrCodeForHTTPStatus(statusCode)
}

func (e Error) sendErrorCode() string {
	switch e.ErrorSubcode {
	case subcodeUserUnavailable, subcodeNoMatchingUser, subcodeOutsideWindow, subcodeOutsideWindowV1:
		return sharedmodel.SendErrRecipientUnavailable
	case subcodeFetchFailed, subcodeUploadFailed:
		return sharedmodel.SendErrFileUnavailable
	}

	switch e.Code {
	case codeUserUnavailable:
		return sharedmodel.SendErrRecipientUnavailable
	// An expired or revoked page token breaks the whole channel.
	case codeAccessToken:
		return sharedmodel.SendErrChannelUnavailable
	// Rate limits and transient platform errors (code 1 is Graph's "unknown
	// error, retry later"): the channel is degraded, not the message invalid.
	case codeUnknown, codeServiceUnavailable, codeTooManyCalls, codeUserRequestLimit, codePageRequestLimit,
		codeRateLimit, codeTemporaryFailure, codeMessengerRateLimit:
		return sharedmodel.SendErrChannelUnavailable
	}

	// Attachment problems share code 100 with every other bad parameter, and
	// Meta does not document stable subcodes for size or type, so the message
	// is the only discriminator.
	msg := strings.ToLower(e.Message + " " + e.ErrorUserMsg)

	switch {
	case strings.Contains(msg, "size") && (strings.Contains(msg, "exceed") || strings.Contains(msg, "too large") || strings.Contains(msg, "limit")):
		return sharedmodel.SendErrFileTooLarge
	case strings.Contains(msg, "failed to fetch"), strings.Contains(msg, "upload attachment failure"):
		return sharedmodel.SendErrFileUnavailable
	case strings.Contains(msg, "not supported") && (strings.Contains(msg, "attachment") || strings.Contains(msg, "media") || strings.Contains(msg, "type")):
		return sharedmodel.SendErrUnsupportedMedia
	default:
		return ""
	}
}
