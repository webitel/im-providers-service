package webhook

import "github.com/webitel/webitel-go-kit/pkg/errors"

var (
	WebhookErrDisablled = errors.NotFound("zero enabled gates found for coresponding whatsapp business account phone id")
)
