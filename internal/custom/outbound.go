package custom

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	contactv1 "github.com/webitel/im-providers-service/gen/go/contact/v1"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
	custommodel "github.com/webitel/im-providers-service/internal/custom/model"
)

const webitelSenderType = "webitel"

func (p *customProvider) SendText(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	return p.send(ctx, req, func(msg *wireMessage) { msg.Text = req.Text })
}

func (p *customProvider) SendImage(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	if len(req.Images) == 0 || req.Images[0] == nil {
		return nil, errors.New("custom: image payload missing")
	}

	img := req.Images[0]

	return p.send(ctx, req, func(msg *wireMessage) {
		msg.Text = req.Text
		msg.File = &wireFile{URL: img.URL, Mime: img.MimeType, Size: img.Size, Name: img.FileName}
	})
}

func (p *customProvider) SendDocument(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	if len(req.Documents) == 0 || req.Documents[0] == nil {
		return nil, errors.New("custom: document payload missing")
	}

	doc := req.Documents[0]

	return p.send(ctx, req, func(msg *wireMessage) {
		msg.Text = req.Text
		msg.File = &wireFile{URL: doc.URL, Mime: doc.MimeType, Size: doc.Size, Name: doc.FileName}
	})
}

func (p *customProvider) SendInteractive(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	menu := buildMenu(req.Interactive)
	if menu == nil {
		return nil, errors.New("custom: interactive payload carries no buttons")
	}

	return p.send(ctx, req, func(msg *wireMessage) {
		msg.Text = req.Text
		msg.Menu = menu
	})
}

func (p *customProvider) SendLocation(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	if req.Location == nil {
		return nil, errors.New("custom: location payload missing")
	}

	return p.send(ctx, req, func(msg *wireMessage) {
		msg.Text = req.Text
		msg.Location = &wireLocation{
			Lat:     req.Location.Latitude,
			Lon:     req.Location.Longitude,
			Title:   req.Location.Name,
			Address: req.Location.Address,
		}
	})
}

func (p *customProvider) SendContact(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	if req.Contact == nil {
		return nil, errors.New("custom: contact payload missing")
	}

	return p.send(ctx, req, func(msg *wireMessage) {
		msg.Text = req.Text
		msg.Contact = &wireContact{
			Name:  req.Contact.Name,
			Phone: req.Contact.PhoneNumber,
			Email: req.Contact.Email,
		}
	})
}

// send addresses one outbound message. A recipient we have no conversation for
// has never written to us, so the message is an operator-initiated one and goes
// out as a broadcast — the shape the contract reserves for exactly that case.
func (p *customProvider) send(
	ctx context.Context,
	req *sharedmodel.Message,
	fill func(*wireMessage),
) (*sharedmodel.MessageResponse, error) {
	messageID := newMessageID(req.ID)

	gate, chat, err := p.resolveChat(ctx, req.GateID, req.To)
	if err != nil {
		if errors.Is(err, custommodel.ErrChatUnknown) {
			return p.broadcast(ctx, req, messageID)
		}

		return nil, err
	}

	msg := &wireMessage{
		ID:      messageID,
		ChatID:  chat.ChatID,
		Sender:  outboundSender(req),
		Date:    nowMilli(),
		ReplyTo: req.ReplyToExternalID,
	}
	fill(msg)

	return p.deliver(ctx, gate, chat.ChatID, messageID, envelope{Message: msg})
}

func (p *customProvider) broadcast(ctx context.Context, req *sharedmodel.Message, messageID string) (*sharedmodel.MessageResponse, error) {
	gate, err := p.fetchGate(ctx, req.GateID)
	if err != nil {
		return nil, err
	}

	sub, err := p.resolveReceiver(ctx, gate, req.To)
	if err != nil {
		return nil, err
	}

	kind, id := splitSub(sub)

	return p.deliver(ctx, gate, sub, messageID, envelope{
		Broadcast: &wireBroadcast{
			EventID:    messageID,
			Recipients: []wireRecipient{{ID: id, Type: kind}},
			Text:       req.Text,
		},
	})
}

// deliver posts one payload and decides what a failure means. A refusal from
// the external system is final and surfaces to the operator now; an
// unreachable endpoint is handed to the retry queue, and the send is reported
// as accepted so the message keeps its reference — the queue marks it failed
// itself once the attempts run out.
func (p *customProvider) deliver(
	ctx context.Context,
	gate *custommodel.CustomGate,
	chatKey string,
	messageID string,
	env envelope,
) (*sharedmodel.MessageResponse, error) {
	payload, err := json.Marshal(env)
	if err != nil {
		return nil, fmt.Errorf("custom: encode payload: %w", err)
	}

	err = p.api.post(ctx, gate, payload)
	if err == nil {
		return &sharedmodel.MessageResponse{ID: messageID}, nil
	}

	if errors.Is(err, custommodel.ErrCallbackRejected) {
		return nil, err
	}

	p.logger.WarnContext(ctx, "callback delivery failed, queued for retry",
		"gate_id", gate.ID,
		"message_id", messageID,
		"err", err,
	)

	p.retries.enqueue(context.WithoutCancel(ctx), retryTask{
		gate:      gate,
		chatKey:   chatKey,
		payload:   payload,
		messageID: messageID,
		attempt:   0,
		lastErr:   err,
	})

	return &sharedmodel.MessageResponse{ID: messageID}, nil
}

func (p *customProvider) resolveChat(ctx context.Context, gateID string, to sharedmodel.Peer) (*custommodel.CustomGate, *custommodel.Chat, error) {
	gate, err := p.fetchGate(ctx, gateID)
	if err != nil {
		return nil, nil, err
	}

	sub, err := p.resolveReceiver(ctx, gate, to)
	if err != nil {
		return nil, nil, err
	}

	chat, err := p.repo.ChatByRecipient(ctx, gate.ID, sub)
	if err != nil {
		if errors.Is(err, sharedstore.ErrNotFound) {
			return nil, nil, custommodel.ErrChatUnknown
		}

		return nil, nil, err
	}

	return gate, chat, nil
}

// resolveReceiver turns the recipient the core addresses — a Webitel contact id
// — back into the external subject the channel knows.
func (p *customProvider) resolveReceiver(ctx context.Context, gate *custommodel.CustomGate, to sharedmodel.Peer) (string, error) {
	contactID := to.Sub
	if to.ID != uuid.Nil {
		contactID = to.ID.String()
	}

	if contactID == "" {
		return "", errors.New("custom: recipient is empty")
	}

	if _, err := uuid.Parse(contactID); err != nil {
		return contactID, nil
	}

	if sub, ok, _ := p.receiverCache.Get(ctx, contactID); ok {
		return sub, nil
	}

	authCtx := withGatewayIdentity(ctx, gate)

	resp, err := p.contactClient.SearchContact(authCtx, &contactv1.SearchContactRequest{
		Ids: []string{contactID},
	})
	if err != nil {
		return "", fmt.Errorf("resolve custom subject for %s: %w", contactID, err)
	}

	items := resp.GetContacts()
	if len(items) == 0 || items[0].GetSubject() == "" {
		return "", fmt.Errorf("resolve custom subject for %s: contact not found or has no subject", contactID)
	}

	sub := items[0].GetSubject()
	_ = p.receiverCache.Set(ctx, contactID, sub)

	return sub, nil
}

// outboundSender carries the name the client should see. The core already
// resolves it to the operator's chat name, keeping the real name for Webitel's
// own history.
func outboundSender(req *sharedmodel.Message) *wireSender {
	return &wireSender{
		ID:   req.From.Sub,
		Type: webitelSenderType,
		Name: req.SenderName,
	}
}

// splitSub reverses the "{type}|{id}" subject back into the pair the channel
// uses to address its own user.
func splitSub(sub string) (kind, id string) {
	if idx := strings.Index(sub, "|"); idx >= 0 {
		return sub[:idx], sub[idx+1:]
	}

	return "", sub
}

func newMessageID(id uuid.UUID) string {
	if id != uuid.Nil {
		return id.String()
	}

	return uuid.NewString()
}
