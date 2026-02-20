package amqp

import (
	"encoding/json"
	"runtime/debug"
	"strings"

	"github.com/ThreeDotsLabs/watermill/message"
)

// [VALIDATOR_INTERFACE]
// Validator is an interface that DTOs can implement to provide their own validation logic.
type Validator interface {
	Validate() error
}

type DomainHandler[T any] func(msg *message.Message, providerName string, payload *T) error

func Bind[T any](h *MessageHandler, fn DomainHandler[T]) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		defer func() {
			if r := recover(); r != nil {
				h.logger.Error("PANIC_RECOVERED",
					"err", r,
					"stack", string(debug.Stack()),
					"msg_id", msg.UUID)
			}
		}()

		pType, ok := resolveProviderType(msg)
		if !ok {
			h.logger.Warn("ROUTING_FAILED: unknown_provider", "msg_id", msg.UUID)
			return nil
		}

		payload := new(T)
		if err := json.Unmarshal(msg.Payload, payload); err != nil {
			h.logger.Error("DECODE_FAILED", "err", err, "msg_id", msg.UUID)
			return nil
		}

		// [VALIDATION_CHECK]
		// We use type assertion to check if our DTO implements Validator.
		if v, ok := any(payload).(Validator); ok {
			if err := v.Validate(); err != nil {
				h.logger.Warn("VALIDATION_FAILED",
					"provider", pType,
					"err", err,
					"msg_id", msg.UUID)
				return nil // ACK: terminal state for invalid data
			}
		}

		return fn(msg, pType, payload)
	}
}

func resolveProviderType(msg *message.Message) (string, bool) {
	rk := msg.Metadata.Get("x-routing-key")
	if rk == "" {
		rk = msg.Metadata.Get("routing_key")
	}
	parts := strings.Split(rk, ".")
	if len(parts) >= 3 {
		return parts[2], true
	}
	return "", false
}
