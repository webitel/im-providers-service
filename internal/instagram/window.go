package instagram

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// Instagram messaging windows.
//
// After a user-initiated message, the business has 24 hours to send a RESPONSE message.
// After that window closes, messages must use MESSAGE_TAG with tag=HUMAN_AGENT (valid for up to 7 days,
// human-sent only).
// https://developers.instagram.com/docs/instagram-api/reference/ig-user/messages
const messagingWindowTTL = 24 * time.Hour //nolint:unused // called by outbound.SendText via withinMessageWindow

func inboundTimestampKey(gateID, igsid string) string { //nolint:unused // called by recordInboundTimestamp and withinMessageWindow
	return "instagram:inbound:" + gateID + ":" + igsid
}

// recordInboundTimestamp stores the timestamp of the last user-initiated message
// in Redis with 24h TTL. Called during inbound message processing.
func (p *instagramProvider) recordInboundTimestamp(ctx context.Context, gateID, igsid string) { //nolint:unused // called from webhook.go
	if p.rdb == nil || gateID == "" || igsid == "" {
		return
	}

	if err := p.rdb.Set(ctx, inboundTimestampKey(gateID, igsid), time.Now().Unix(), messagingWindowTTL).Err(); err != nil {
		p.logger.WarnContext(ctx, "instagram inbound timestamp not stored", "gate_id", gateID, "igsid", igsid, "err", err)
	}
}

// withinMessageWindow checks if the last inbound message from the user was within
// the 24-hour messaging window. Returns true if within window, false otherwise.
func (p *instagramProvider) withinMessageWindow(ctx context.Context, gateID, igsid string) bool {
	if p.rdb == nil || gateID == "" || igsid == "" {
		return false
	}

	val, err := p.rdb.Get(ctx, inboundTimestampKey(gateID, igsid)).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			p.logger.WarnContext(ctx, "instagram message window lookup failed", "gate_id", gateID, "igsid", igsid, "err", err)
		}

		return false
	}

	var timestamp int64

	_, err = readInt64(val, &timestamp)
	if err != nil {
		p.logger.WarnContext(ctx, "instagram inbound timestamp parse failed", "gate_id", gateID, "igsid", igsid, "err", err)

		return false
	}

	elapsed := time.Since(time.Unix(timestamp, 0))

	return elapsed < messagingWindowTTL
}

// readInt64 parses val as an int64. Returns the parsed value and any parsing error.
func readInt64(val string, dst *int64) (int64, error) {
	var result int64

	for i := range len(val) { //nolint:intrange // Go 1.22+ syntax
		c := val[i]
		if c < '0' || c > '9' {
			return 0, errors.New("invalid int64 format")
		}

		result = result*10 + int64(c-'0')
	}

	*dst = result

	return result, nil
}
