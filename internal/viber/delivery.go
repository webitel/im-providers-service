package viber

import (
	"context"
	"time"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

// Viber delivery callbacks. "delivered" and "seen" arrive for every outbound message,
// "failed" only when the platform gave up on one.
// https://developers.viber.com/docs/api/rest-bot-api/#callbacks
const (
	deliveryDelivered = sharedmodel.DeliveryDelivered
	deliveryRead      = sharedmodel.DeliveryRead
	deliveryFailed    = sharedmodel.DeliveryFailed
)

func (p *viberProvider) reportDelivery(ctx context.Context, evt *inboundEvent, status sharedmodel.DeliveryStatus) error {
	token := evt.MessageToken.String()
	if token == "" {
		return nil
	}

	gate, err := p.gateFor(ctx)
	if err != nil || gate == nil {
		return err
	}

	if status == deliveryFailed {
		if evt.UserID != "" {
			p.setSubscription(ctx, gate.ID, evt.UserID, false)
		}

		// The send ref is saved under the message token on a successful send,
		// so the failure resolves to the agent's message in im-thread. This is
		// the only failure record: UpdateMessageDelivery below just stamps
		// failed_at/failed_reason on message_external_ids and emits nothing.
		p.status.FailedByProviderID(ctx, gate.ID, token, callbackTime(evt.Timestamp), failedCallbackCode(evt.Desc), evt.Desc)
	}

	sub := ""
	if actor := evt.actor(); actor != nil {
		sub = actor.ID
	}

	report := &sharedmodel.MessageDeliveryReport{
		GateID:     gate.ID,
		ExternalID: token,
		Status:     status,
		Reason:     evt.Desc,
		DomainID:   gate.DomainID,
		Sub:        sub,
	}

	if err := p.messenger.UpdateMessageDelivery(ctx, report); err != nil {
		p.logger.WarnContext(ctx, "viber delivery report not recorded",
			"gate_id", gate.ID,
			"token", token,
			"status", status.String(),
			"err", err,
		)
	}

	return nil
}

// callbackTime converts the callback epoch-milliseconds timestamp, falling
// back to now when Viber omits it.
func callbackTime(ms int64) time.Time {
	if ms <= 0 {
		return time.Now()
	}

	return time.UnixMilli(ms)
}
