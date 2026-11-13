package common

import (
	"strings"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

// WhatsApp Cloud API error codes shared by synchronous send answers and
// "failed" status webhooks.
// https://developers.facebook.com/docs/whatsapp/cloud-api/support/error-codes
const (
	codeServiceTemporary  = 2
	codeTooManyCalls      = 4
	codeUserRequestLimit  = 17
	codeAccessToken       = 190
	codeThroughputLimit   = 80007
	codeRateLimit         = 130429
	codeGeneric           = 131000
	codeServiceOverloaded = 131016
	codeRecipientIsSender = 131021
	codeUndeliverable     = 131026
	codeReEngagement      = 131047
	codeSpamRateLimit     = 131048
	codeUnsupportedType   = 131051
	codeMediaDownload     = 131052
	codeMediaUpload       = 131053
	codePairRateLimit     = 131056
	codeRegisterRateLimit = 133016
)

// SendErrorCode maps a WhatsApp error code (plus its details text) to a
// normalized code, "" when the code is not specific: callers then fall back
// to the gRPC code of the error (sync sends) or send_failed (webhooks).
// 131053 covers every media upload problem, so the size variant is told
// apart by its details.
func SendErrorCode(code int, details string) string {
	switch code {
	case codeMediaUpload:
		if strings.Contains(strings.ToLower(details), "size") {
			return sharedmodel.SendErrFileTooLarge
		}

		return sharedmodel.SendErrFileUnavailable
	case codeMediaDownload:
		return sharedmodel.SendErrFileUnavailable
	case codeUnsupportedType:
		return sharedmodel.SendErrUnsupportedMedia
	case codeUndeliverable, codeReEngagement, codeRecipientIsSender:
		return sharedmodel.SendErrRecipientUnavailable
	// Rate limits and temporary platform failures: retrying later can succeed.
	// An expired or revoked access token breaks the whole channel, like a
	// revoked Telegram bot token.
	case codeAccessToken:
		return sharedmodel.SendErrChannelUnavailable
	case codeServiceTemporary, codeTooManyCalls, codeUserRequestLimit, codeThroughputLimit, codeRateLimit,
		codeGeneric, codeServiceOverloaded, codeSpamRateLimit, codePairRateLimit, codeRegisterRateLimit:
		return sharedmodel.SendErrChannelUnavailable
	default:
		return ""
	}
}
