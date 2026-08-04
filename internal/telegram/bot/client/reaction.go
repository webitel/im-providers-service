package client

import (
	"context"
	"encoding/json"
)

const (
	setMessageReactionMethod = "setMessageReaction"
)

// ReactionEmoji represents a single emoji reaction.
type ReactionEmoji struct {
	Type  string `json:"type"`
	Emoji string `json:"emoji"`
}

// ReactionRequest is the request to set a reaction on a Telegram message.
type ReactionRequest struct {
	ChatID    int64           `json:"chat_id"`
	MessageID int64           `json:"message_id"`
	Reaction  []ReactionEmoji `json:"reaction,omitempty"`
	IsBig     bool            `json:"is_big,omitempty"`
}

// SetMessageReaction sets or clears an emoji reaction on a Telegram message.
// An empty reaction slice clears all reactions.
func (c *Client) SetMessageReaction(ctx context.Context, token string, req *ReactionRequest) error {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return err
	}

	// setMessageReaction returns no result, just pass nil for the output.
	if err := c.callMethod(ctx, setMessageReactionMethod, token, reqBody, nil); err != nil {
		return err
	}

	return nil
}
