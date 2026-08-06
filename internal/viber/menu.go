package viber

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	vibmodel "github.com/webitel/im-providers-service/internal/viber/model"
)

const activeMenuTTL = 24 * time.Hour

type menuButton struct {
	Code string `json:"code"`
	Data string `json:"data"`
}

type activeMenu struct {
	MessageID string                `json:"message_id"`
	Buttons   map[string]menuButton `json:"buttons"`
}

func activeMenuKey(gateID, userID string) string {
	return "viber:menu:" + gateID + ":" + userID
}

func menuFor(messageID string, interactive *sharedmodel.Interactive) activeMenu {
	menu := activeMenu{MessageID: messageID, Buttons: map[string]menuButton{}}

	for _, b := range interactiveButtons(interactive) {
		payload, ok := replyPayload(b)
		if !ok || payload == "" {
			continue
		}

		code := b.ID
		if code == "" {
			code = payload
		}

		menu.Buttons[payload] = menuButton{Code: code, Data: payload}
	}

	return menu
}

func matchTap(menu activeMenu, msg *inboundMessage) (*menuButton, string, bool) {
	if msg == nil || msg.Text == "" {
		return nil, "", false
	}

	btn, ok := menu.Buttons[msg.Text]
	if !ok {
		return nil, "", false
	}

	messageID := menu.MessageID
	if msg.TrackingData != "" {
		messageID = msg.TrackingData
	}

	return &btn, messageID, true
}

func (p *viberProvider) rememberMenu(ctx context.Context, gateID, userID, messageID string, interactive *sharedmodel.Interactive) {
	if p.rdb == nil || gateID == "" || userID == "" || interactive == nil {
		return
	}

	menu := menuFor(messageID, interactive)
	if len(menu.Buttons) == 0 {
		return
	}

	payload, err := json.Marshal(menu)
	if err != nil {
		return
	}

	if err := p.rdb.Set(ctx, activeMenuKey(gateID, userID), payload, activeMenuTTL).Err(); err != nil {
		p.logger.WarnContext(ctx, "viber active menu not stored", "gate_id", gateID, "err", err)
	}
}

func (p *viberProvider) resolveMenuTap(ctx context.Context, gateID, userID string, msg *inboundMessage) (*menuButton, string, bool) {
	if p.rdb == nil || gateID == "" || userID == "" || msg == nil || msg.Text == "" {
		return nil, "", false
	}

	raw, err := p.rdb.Get(ctx, activeMenuKey(gateID, userID)).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			p.logger.WarnContext(ctx, "viber active menu lookup failed", "gate_id", gateID, "err", err)
		}

		return nil, "", false
	}

	var menu activeMenu
	if err := json.Unmarshal([]byte(raw), &menu); err != nil {
		return nil, "", false
	}

	return matchTap(menu, msg)
}

func (p *viberProvider) forwardMenuTap(ctx context.Context, gate *vibmodel.ViberGate, peers peerPair, btn *menuButton, messageID string) error {
	return p.messenger.SendInteractiveCallback(ctx, &sharedmodel.SendInteractiveCallbackRequest{
		From:         peers.from,
		To:           peers.to,
		DomainID:     gate.DomainID,
		InReplyTo:    messageID,
		ButtonCode:   btn.Code,
		CallbackData: btn.Data,
	})
}
