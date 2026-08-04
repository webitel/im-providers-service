package bot

import (
	"context"
	"strconv"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	coremodel "github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/im-providers-service/internal/telegram/bot/model"
)

// handleCallbackQuery processes a callback query update from Telegram.
func (p *Provider) handleCallbackQuery(ctx context.Context, gate *model.Gate, query *model.CallbackQuery) error {
	if gate == nil {
		return errors.Internal("gate required to process callback query update")
	}
	if query == nil {
		return errors.Internal("query required to process callback query update")
	}
	if query.Data == nil {
		return errors.Internal("data required to process callback query update")
	}

	data := *query.Data
	from, to, err := p.getCallbackQueryRecipients(ctx, gate, query)
	if err != nil {
		return err
	}
	req := &coremodel.SendInteractiveCallbackRequest{
		From:         from,
		To:           to,
		DomainID:     gate.DC,
		CallbackData: data,
	}
	if query.Message != nil {
		req.InReplyTo = strconv.FormatInt(query.Message.MessageID, 10)
	}

	err = p.coreMessageClient.SendInteractiveCallback(withVia(ctx, gate), req)
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) getCallbackQueryRecipients(ctx context.Context, gate *model.Gate, query *model.CallbackQuery) (from, to coremodel.Peer, err error) {
	if query.From == nil {
		return coremodel.Peer{}, coremodel.Peer{}, errors.Internal("from required to process callback query update")
	}
	return from, to, nil
}
