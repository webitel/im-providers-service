package viberbm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
	vibbmmodel "github.com/webitel/im-providers-service/internal/viberbm/model"
)

// receivedAtLayout is InfoBip's timestamp format for MO/DLR/Seen reports.
const receivedAtLayout = "2006-01-02T15:04:05.999-0700"

// inboundBatch is one InfoBip forward: {"results":[...]}. MO message, DLR and
// Seen share the endpoint and are told apart by which fields are populated.
type inboundBatch struct {
	Results []inboundResult `json:"results"`
}

//nolint:tagliatelle // InfoBip omni-channel wire format is camelCase
type inboundResult struct {
	// From is the end-user MSISDN on MO messages; UnmarshalJSON also accepts the
	// `sender` alias some docs use.
	// https://www.infobip.com/docs/api/channels/viber/viber-business-messages/inbound-message/receive-viber-business-message
	From       string          `json:"from"`
	To         string          `json:"to"`
	MessageID  string          `json:"messageId"`
	PushName   string          `json:"pushName"`
	ReceivedAt string          `json:"receivedAt"`
	DoneAt     string          `json:"doneAt"`
	SeenAt     string          `json:"seenAt"`
	Message    *inboundMessage `json:"message"`
	Status     *inboundStatus  `json:"status"`
}

// UnmarshalJSON tolerates `sender` as an alias for `from`.
func (r *inboundResult) UnmarshalJSON(data []byte) error {
	type alias inboundResult

	aux := &struct {
		*alias

		Sender string `json:"sender"`
	}{alias: (*alias)(r)}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	if r.From == "" && aux.Sender != "" {
		r.From = aux.Sender
	}

	return nil
}

//nolint:tagliatelle // InfoBip omni-channel wire format is camelCase
type inboundMessage struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	URL      string `json:"url"`
	FileName string `json:"fileName"`
	Caption  string `json:"caption"`
}

//nolint:tagliatelle // InfoBip omni-channel wire format is camelCase
type inboundStatus struct {
	GroupID     int    `json:"groupId"`
	GroupName   string `json:"groupName"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (p *viberBMProvider) HandleWebhook(ctx context.Context, data []byte) error {
	var batch inboundBatch
	if err := json.Unmarshal(data, &batch); err != nil {
		return fmt.Errorf("viber_bm: parse webhook: %w", err)
	}

	uri := p.webhookURI(ctx)

	// All results on a uri belong to one gate (see resolveGate). Unknown/disabled
	// gate is a permanent 200; an infra failure returns non-nil so InfoBip retries.
	gate, err := p.resolveGate(ctx, uri)
	if err != nil {
		if errors.Is(err, sharedstore.ErrNotFound) {
			return nil
		}

		p.logger.ErrorContext(ctx, "viber_bm: gate resolve failed", "err", err)

		return err
	}

	if !gate.Enabled {
		return nil
	}

	var retryErr error

	for i := range batch.Results {
		res := &batch.Results[i]

		if err := p.processResult(ctx, gate, res); err != nil {
			p.logger.ErrorContext(ctx, "viber_bm result dropped", "message_id", res.MessageID, "err", err)
			retryErr = err
		}
	}

	return retryErr
}

func (p *viberBMProvider) processResult(ctx context.Context, gate *vibbmmodel.ViberBMGate, res *inboundResult) error {
	switch {
	case res.Message != nil:
		return p.processMessage(ctx, gate, res)
	case isTerminalFailure(res.Status):
		// A terminal failure wins over a stray seenAt so a REJECTED message is
		// reported as failed, not read.
		p.processReceipt(ctx, gate, res)

		return nil
	case res.SeenAt != "" || isSeenStatus(res.Status):
		p.processSeen(ctx, gate, res)

		return nil
	case res.Status != nil:
		p.processReceipt(ctx, gate, res)

		return nil
	default:
		return nil
	}
}

// processMessage handles an inbound MO message: dedup → sync contact → route.
func (p *viberBMProvider) processMessage(ctx context.Context, gate *vibbmmodel.ViberBMGate, res *inboundResult) error {
	if res.From == "" {
		return nil
	}

	// syncContact is idempotent, so it runs before the dedup mark: a failure
	// here leaves the messageId unmarked and InfoBip's redelivery can retry.
	if _, err := p.syncContact(ctx, gate, res.From, res.PushName); err != nil {
		return fmt.Errorf("sync contact [from=%s]: %w", res.From, err)
	}

	if messageSeen(ctx, p.rdb, res.MessageID) {
		p.logger.DebugContext(ctx, "duplicate viber_bm event skipped", "message_id", res.MessageID)

		return nil
	}

	// Do NOT set Via on the bot peer: gate identity travels via the
	// x-webitel-via header (coreMessengerFor); tagging the bot would make
	// thread-service address outbound replies to the bot, not the customer.
	peers := peerPair{
		from: sharedmodel.Peer{Sub: res.From, Iss: gate.Peer.Iss},
		to:   sharedmodel.Peer{Sub: gate.Peer.Sub, Iss: gate.Peer.Iss},
	}

	msg := res.Message
	switch strings.ToUpper(msg.Type) {
	case contentTypeText:
		if _, err := p.coreMessengerFor(gate).SendText(ctx, &sharedmodel.SendTextRequest{
			DomainID:   gate.DomainID,
			From:       peers.from,
			To:         peers.to,
			Body:       msg.Text,
			ExternalID: res.MessageID,
		}); err != nil {
			p.logger.ErrorContext(ctx, "viber_bm send text failed", "message_id", res.MessageID, "err", err)
		}
	default:
		p.routeMedia(ctx, gate, peers, res)
	}

	return nil
}

// routeMedia downloads inbound media and forwards it as image or document.
func (p *viberBMProvider) routeMedia(ctx context.Context, gate *vibbmmodel.ViberBMGate, peers peerPair, res *inboundResult) {
	msg := res.Message
	if msg.URL == "" {
		p.logger.WarnContext(ctx, "viber_bm inbound media without url", "type", msg.Type, "message_id", res.MessageID)

		return
	}

	name := msg.FileName
	if name == "" {
		name = fmt.Sprintf("viber_bm_%d", time.Now().Unix())
	}

	media, err := p.downloadInboundMedia(ctx, gate, msg.URL, name)
	if err != nil {
		// The InfoBip CDN URL can embed a signed token, so it is not logged.
		p.logger.ErrorContext(ctx, "viber_bm media sync failed", "message_id", res.MessageID, "err", err)

		return
	}

	if media.isImage {
		if _, err := p.coreMessengerFor(gate).SendImage(ctx, &sharedmodel.SendImageRequest{
			DomainID: gate.DomainID,
			From:     peers.from,
			To:       peers.to,
			Image: sharedmodel.ImageRequest{
				Body: msg.Caption,
				Images: []*sharedmodel.Image{{
					ID:       media.id,
					FileName: name,
					MimeType: media.mimeType,
					Size:     media.size,
				}},
			},
			ExternalID: res.MessageID,
		}); err != nil {
			p.logger.ErrorContext(ctx, "viber_bm send image failed", "message_id", res.MessageID, "err", err)
		}

		return
	}

	if _, err := p.coreMessengerFor(gate).SendDocument(ctx, &sharedmodel.SendDocumentRequest{
		DomainID: gate.DomainID,
		From:     peers.from,
		To:       peers.to,
		Document: sharedmodel.DocumentRequest{
			Body: msg.Caption,
			Documents: []*sharedmodel.Document{{
				ID:       media.id,
				FileName: name,
				MimeType: media.mimeType,
				Size:     media.size,
			}},
		},
		ExternalID: res.MessageID,
	}); err != nil {
		p.logger.ErrorContext(ctx, "viber_bm send document failed", "message_id", res.MessageID, "err", err)
	}
}

// processReceipt maps an InfoBip DLR to a delivery/failure report, correlated
// by our outbound-stamped messageId.
func (p *viberBMProvider) processReceipt(ctx context.Context, gate *vibbmmodel.ViberBMGate, res *inboundResult) {
	if res.MessageID == "" || res.Status == nil {
		return
	}

	at := parseInfobipTime(res.DoneAt)

	switch strings.ToUpper(res.Status.GroupName) {
	case "DELIVERED":
		p.status.DeliveredByProviderIDs(ctx, gate.ID, []string{res.MessageID}, at)
	case "REJECTED", "EXPIRED", "UNDELIVERABLE":
		p.status.FailedByProviderID(ctx, gate.ID, res.MessageID, at, res.Status.Name, res.Status.Description)
	default:
		// PENDING and other transient states are ignored; a terminal DLR follows.
	}
}

// processSeen maps an InfoBip Seen report to a read-up-to receipt.
// https://www.infobip.com/docs/api/channels/viber/viber-business-messages/inbound-message/receive-viber-business-message
func (p *viberBMProvider) processSeen(ctx context.Context, gate *vibbmmodel.ViberBMGate, res *inboundResult) {
	// A Seen is about an outbound message the customer read, so the customer is
	// `to` (a Seen's `from` is our own sender name). Normalise symmetrically with
	// the outbound-stored recipient_id, else the watermark finds no refs.
	if res.To == "" {
		return
	}

	p.status.ReadUpTo(ctx, gate.ID, normalizeMSISDN(res.To), parseInfobipTime(res.SeenAt))
}

func isSeenStatus(s *inboundStatus) bool {
	return s != nil && strings.EqualFold(s.GroupName, "SEEN")
}

// isTerminalFailure reports whether a DLR is a terminal non-delivery, which
// must be reported as failed even if a seenAt is also present on the result.
func isTerminalFailure(s *inboundStatus) bool {
	if s == nil {
		return false
	}

	switch strings.ToUpper(s.GroupName) {
	case "REJECTED", "EXPIRED", "UNDELIVERABLE":
		return true
	default:
		return false
	}
}

// parseInfobipTime falls back to now on an unparseable stamp: a small clock
// skew beats dropping the receipt.
func parseInfobipTime(s string) time.Time {
	if t, err := time.Parse(receivedAtLayout, s); err == nil {
		return t
	}

	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}

	return time.Now()
}
