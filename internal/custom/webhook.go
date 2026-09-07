package custom

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc/metadata"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
	custommodel "github.com/webitel/im-providers-service/internal/custom/model"
)

const sourceVariable = "source"

func (p *customProvider) HandleWebhook(ctx context.Context, data []byte) error {
	var evt envelope
	if err := json.Unmarshal(data, &evt); err != nil {
		return fmt.Errorf("custom: unparsable payload: %w", err)
	}

	switch {
	case evt.Message != nil:
		return p.handleMessage(ctx, evt.Message)
	case evt.Status != nil:
		return p.handleStatus(ctx, evt.Status)
	case evt.Broadcast != nil:
		return p.handleBroadcastResult(ctx, evt.Broadcast)
	default:
		return errors.New("custom: payload carries no message, status or broadcast")
	}
}

// WebhookResponse renders the response body the contract prescribes
func (p *customProvider) WebhookResponse(handleErr error) (string, []byte) {
	resp := wireResponse{Success: true}
	if handleErr != nil {
		resp = wireResponse{Error: handleErr.Error()}
	}

	body, err := json.Marshal(resp)
	if err != nil {
		return "application/json", []byte(`{"success":false,"error":"internal"}`)
	}

	return "application/json", body
}

func (p *customProvider) gateFor(ctx context.Context) (*custommodel.CustomGate, error) {
	gate, err := p.resolveGate(ctx, p.webhookURI(ctx))
	if err != nil {
		return nil, err
	}

	if gate == nil || !gate.Enabled {
		p.logger.WarnContext(ctx, "custom webhook for a missing or disabled gate, dropping")

		return nil, nil //nolint:nilnil // documented "drop it" signal
	}

	return gate, nil
}

func (p *customProvider) handleMessage(ctx context.Context, msg *wireMessage) error {
	if msg.Sender == nil || msg.Sender.ID == "" {
		return errors.New("custom: message.sender.id is required")
	}

	if msg.ChatID == "" {
		return errors.New("custom: message.chatId is required")
	}

	gate, err := p.gateFor(ctx)
	if err != nil || gate == nil {
		return err
	}

	if messageSeen(ctx, p.rdb, gate.ID, msg.ID) {
		p.logger.DebugContext(ctx, "duplicate custom message skipped", "gate_id", gate.ID, "external_id", msg.ID)

		return nil
	}

	if _, err := p.syncContact(ctx, gate, msg.Sender); err != nil {
		return fmt.Errorf("sync contact [id=%s]: %w", msg.Sender.ID, err)
	}

	sub := contactSub(msg.Sender)

	firstOfChat := p.isNewChat(ctx, gate.ID, msg.ChatID)
	if err := p.repo.UpsertChat(ctx, gate.ID, msg.ChatID, sub); err != nil {
		p.logger.ErrorContext(ctx, "failed to record custom chat", "gate_id", gate.ID, "chat_id", msg.ChatID, "err", err)
	}

	in := &inbound{
		gate: gate,
		peers: peerPair{
			from: sharedmodel.Peer{Sub: sub, Iss: gate.Peer.Iss, Name: senderName(msg.Sender)},
			to:   sharedmodel.Peer{Sub: gate.Peer.Sub, Iss: gate.Peer.Iss},
		},
		msg:        msg,
		externalID: msg.ID,
	}

	if firstOfChat {
		in.variables = threadVariables(msg)
	}

	sendCtx := withProviderVia(ctx, gate.DomainID, sub, gate.ID)
	p.routeMessage(sendCtx, in)

	return nil
}

func (p *customProvider) isNewChat(ctx context.Context, gateID, chatID string) bool {
	_, err := p.repo.ChatByID(ctx, gateID, chatID)

	return errors.Is(err, sharedstore.ErrNotFound)
}

func threadVariables(msg *wireMessage) map[string]string {
	vars := make(map[string]string, len(msg.Metadata)+1)
	for k, v := range msg.Metadata {
		vars[k] = v
	}

	if msg.Sender != nil && msg.Sender.Type != "" {
		if _, taken := vars[sourceVariable]; !taken {
			vars[sourceVariable] = msg.Sender.Type
		}
	}

	if len(vars) == 0 {
		return nil
	}

	return vars
}

func senderName(s *wireSender) string {
	if s == nil {
		return ""
	}

	if s.Name != "" {
		return s.Name
	}

	if s.Nickname != "" {
		return s.Nickname
	}

	return s.ID
}

// handleStatus records what the external system reports about a message we sent
func (p *customProvider) handleStatus(ctx context.Context, in *wireStatus) error {
	if in.MessageID == "" {
		return errors.New("custom: status.messageId is required")
	}

	gate, err := p.gateFor(ctx)
	if err != nil || gate == nil {
		return err
	}

	at := time.Now()
	if in.At > 0 {
		at = time.UnixMilli(in.At)
	}

	switch in.Status {
	case "delivered":
		p.status.DeliveredByProviderIDs(ctx, gate.ID, []string{in.MessageID}, at)
	case "read", "seen":
		p.status.ReadByProviderID(ctx, gate.ID, in.MessageID, at)
	case "failed":
		p.status.FailedByProviderID(ctx, gate.ID, in.MessageID, at, "external_failed", in.Reason)
	default:
		return fmt.Errorf("custom: unknown status %q", in.Status)
	}

	return nil
}

// handleBroadcastResult records the asynchronous outcome of an
// operator-initiated message. The contract only reports failures: recipients
// present in the reply are the ones that did not receive it.
func (p *customProvider) handleBroadcastResult(ctx context.Context, in *wireBroadcast) error {
	if in.EventID == "" {
		return errors.New("custom: broadcast.eventId is required")
	}

	gate, err := p.gateFor(ctx)
	if err != nil || gate == nil {
		return err
	}

	if len(in.Recipients) == 0 {
		return nil
	}

	for _, recipient := range in.Recipients {
		p.logger.WarnContext(ctx, "custom broadcast delivery failed",
			"gate_id", gate.ID,
			"event_id", in.EventID,
			"recipient", recipient.ID,
			"err", recipient.Error,
		)

		p.status.FailedByProviderID(ctx, gate.ID, in.EventID, time.Now(), "broadcast_failed", recipient.Error)
	}

	return nil
}

func withProviderVia(ctx context.Context, dc int64, customerSub, gateID string) context.Context {
	return metadata.NewOutgoingContext(ctx, metadata.Pairs(
		"x-webitel-type", "provider",
		"x-webitel-provider", fmt.Sprintf("%d.%s", dc, customerSub),
		"x-webitel-via", gateID,
	))
}
