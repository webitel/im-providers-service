package viber

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// Viber reachability windows.
//
// A bot may only message a user who subscribed to it, with one exception: after a
// conversation_started event the bot may send a single welcome message within five
// minutes even though the user has not subscribed yet.
// https://developers.viber.com/docs/api/rest-bot-api/#conversation-started
const (
	subscriptionTTL = 90 * 24 * time.Hour
	welcomeTTL      = 5 * time.Minute
)

const (
	subscribedFlag   = "1"
	unsubscribedFlag = "0"
)

func subscriptionKey(gateID, userID string) string { return "viber:sub:" + gateID + ":" + userID }
func welcomeKey(gateID, userID string) string      { return "viber:welcome:" + gateID + ":" + userID }

func (p *viberProvider) setSubscription(ctx context.Context, gateID, userID string, subscribed bool) {
	if p.rdb == nil || gateID == "" || userID == "" {
		return
	}

	flag := unsubscribedFlag
	if subscribed {
		flag = subscribedFlag
	}

	if err := p.rdb.Set(ctx, subscriptionKey(gateID, userID), flag, subscriptionTTL).Err(); err != nil {
		p.logger.WarnContext(ctx, "viber subscription state not stored", "gate_id", gateID, "err", err)
	}
}

func (p *viberProvider) openWelcomeWindow(ctx context.Context, gateID, userID string) {
	if p.rdb == nil || gateID == "" || userID == "" {
		return
	}

	if err := p.rdb.Set(ctx, welcomeKey(gateID, userID), subscribedFlag, welcomeTTL).Err(); err != nil {
		p.logger.WarnContext(ctx, "viber welcome window not stored", "gate_id", gateID, "err", err)
	}
}

func (p *viberProvider) unreachable(ctx context.Context, gateID, userID string) bool {
	if p.rdb == nil || gateID == "" || userID == "" {
		return false
	}

	state, err := p.rdb.Get(ctx, subscriptionKey(gateID, userID)).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			p.logger.WarnContext(ctx, "viber subscription state lookup failed", "gate_id", gateID, "err", err)
		}

		return false
	}
	if state != unsubscribedFlag {
		return false
	}

	if open, err := p.rdb.Exists(ctx, welcomeKey(gateID, userID)).Result(); err == nil && open > 0 {
		return false
	}

	return true
}
