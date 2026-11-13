package viberbm

import (
	"errors"
	"strings"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	vibbmmodel "github.com/webitel/im-providers-service/internal/viberbm/model"
)

// sendErrorCode maps a synchronous Viber BM send failure to a normalized
// code, "" when the error says nothing specific.
func sendErrorCode(err error) string {
	switch {
	case errors.Is(err, vibbmmodel.ErrVideoTooLarge):
		return sharedmodel.SendErrFileTooLarge
	case errors.Is(err, vibbmmodel.ErrFileTypeUnsupported),
		errors.Is(err, vibbmmodel.ErrAudioUnsupported),
		errors.Is(err, vibbmmodel.ErrVideoTooLong):
		return sharedmodel.SendErrUnsupportedMedia
	case errors.Is(err, vibbmmodel.ErrVideoPreviewUnavailable):
		return sharedmodel.SendErrFileUnavailable
	case errors.Is(err, vibbmmodel.ErrReceiverNotReachable),
		errors.Is(err, vibbmmodel.ErrSessionWindowRequired):
		return sharedmodel.SendErrRecipientUnavailable
	}

	var httpErr *httpError
	if errors.As(err, &httpErr) {
		return sharedmodel.SendErrCodeForHTTPStatus(httpErr.StatusCode)
	}

	return ""
}

func tagSendError(err error) error {
	return sharedmodel.NewSendError(sendErrorCode(err), err)
}

// InfoBip status names that pin a REJECTED/UNDELIVERABLE outcome on one side.
// The sender/account side (blocked or unpaid sender, no route) is a channel
// problem the agent cannot fix by retrying for another recipient.
// https://www.infobip.com/docs/essentials/response-status-and-error-codes
var (
	recipientStatusNames = map[string]bool{
		"REJECTED_DESTINATION":                true,
		"REJECTED_DESTINATION_NOT_REGISTERED": true,
		"REJECTED_DND":                        true,
		"REJECTED_NOT_VIBER_USER":             true,
	}

	senderStatusNames = map[string]bool{
		"REJECTED_SENDER":                  true,
		"REJECTED_SOURCE":                  true,
		"REJECTED_NOT_ENOUGH_CREDITS":      true,
		"REJECTED_PREPAID_PACKAGE_EXPIRED": true,
		"REJECTED_ROUTE_NOT_AVAILABLE":     true,
		"REJECTED_FLOODING_FILTER":         true,
		"REJECTED_SYSTEM_ERROR":            true,
	}
)

// Description fragments for statuses whose name is generic. A bare "BLOCKED"
// is deliberately absent: "sender blocked" and "user blocked the sender" are
// opposite sides.
var (
	senderMarkers = []string{
		"SENDER BLOCKED", "SENDER IS BLOCKED", "BLOCKED SENDER", "ACCOUNT BLOCKED", "ACCOUNT IS BLOCKED",
		"ACCOUNT SUSPENDED", "SENDER NOT ALLOWED", "NOT ENOUGH CREDITS",
	}

	recipientMarkers = []string{
		"NOT_VIBER_USER", "NOT A VIBER USER", "NO VIBER", "UNSUBSCRIBED", "OPTED OUT", "OPT_OUT",
		"BLOCKED THE SENDER", "BLOCKED BY THE USER", "BLOCKED BY USER", "BLOCKED BY RECIPIENT",
		"USER BLOCKED", "RECIPIENT BLOCKED", "DESTINATION BLOCKED",
	}
)

// statusErrorCode maps an InfoBip message status (sync REJECTED answer or a
// terminal DLR) to a normalized code.
func statusErrorCode(groupName, name, description string) string {
	if strings.EqualFold(groupName, "EXPIRED") {
		return sharedmodel.SendErrRecipientUnavailable
	}

	upperName := strings.ToUpper(name)

	switch {
	case senderStatusNames[upperName]:
		return sharedmodel.SendErrChannelUnavailable
	case recipientStatusNames[upperName]:
		return sharedmodel.SendErrRecipientUnavailable
	}

	text := strings.ToUpper(name + " " + description)

	switch {
	case containsAny(text, senderMarkers):
		return sharedmodel.SendErrChannelUnavailable
	case containsAny(text, recipientMarkers):
		return sharedmodel.SendErrRecipientUnavailable
	default:
		return sharedmodel.SendErrSendFailed
	}
}

func containsAny(s string, fragments []string) bool {
	for _, f := range fragments {
		if strings.Contains(s, f) {
			return true
		}
	}

	return false
}

// statusErrorMessage keeps the raw InfoBip status in the reported message,
// since the code field now carries the normalized value.
func statusErrorMessage(name, description string) string {
	switch {
	case name == "":
		return description
	case description == "":
		return name
	default:
		return name + ": " + description
	}
}
