package custom

import (
	"bytes"
	"encoding/json"
	"strconv"
)

type envelope struct {
	Message   *wireMessage   `json:"message,omitempty"`
	Status    *wireStatus    `json:"status,omitempty"`
	Broadcast *wireBroadcast `json:"broadcast,omitempty"`
}

type wireMessage struct {
	ID     string      `json:"id,omitempty"`
	ChatID string      `json:"chatId,omitempty"`
	Sender *wireSender `json:"sender,omitempty"`
	Date   int64       `json:"date,omitempty"`
	Text   string      `json:"text,omitempty"`
	File   *wireFile   `json:"file,omitempty"`

	// Metadata reaches the flow schema as thread variables, and only on the
	// message that opens a conversation.
	Metadata map[string]string `json:"metadata,omitempty"`

	Location *wireLocation `json:"location,omitempty"`
	Contact  *wireContact  `json:"contact,omitempty"`
	Menu     *wireMenu     `json:"menu,omitempty"`
	Callback *wireCallback `json:"callback,omitempty"`
	ReplyTo  string        `json:"replyTo,omitempty"`
}

type wireSender struct {
	ID       string `json:"id,omitempty"`
	Type     string `json:"type,omitempty"`
	Name     string `json:"name,omitempty"`
	Nickname string `json:"nickname,omitempty"`
}

type wireFile struct {
	URL  string `json:"url,omitempty"`
	Mime string `json:"mime,omitempty"`
	Size int64  `json:"size,omitempty"`
	Name string `json:"name,omitempty"`
}

type wireLocation struct {
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	Title   string  `json:"title,omitempty"`
	Address string  `json:"address,omitempty"`
}

type wireContact struct {
	Name  string `json:"name,omitempty"`
	Phone string `json:"phone,omitempty"`
	Email string `json:"email,omitempty"`
}

type wireMenu struct {
	Text      string          `json:"text,omitempty"`
	SingleUse bool            `json:"singleUse,omitempty"`
	Placement string          `json:"placement,omitempty"`
	Rows      [][]wireButton  `json:"rows,omitempty"`
	Sections  []wireMenuGroup `json:"sections,omitempty"`
}

type wireMenuGroup struct {
	Section string       `json:"section,omitempty"`
	Buttons []wireButton `json:"buttons,omitempty"`
}

type wireButton struct {
	Code   string `json:"code,omitempty"`
	Text   string `json:"text,omitempty"`
	URL    string `json:"url,omitempty"`
	Data   string `json:"data,omitempty"`
	Action string `json:"action,omitempty"`
}

type wireCallback struct {
	Code      string `json:"code,omitempty"`
	MessageID string `json:"messageId,omitempty"`
	Data      string `json:"data,omitempty"`
}

type wireStatus struct {
	ChatID    string `json:"chatId,omitempty"`
	MessageID string `json:"messageId,omitempty"`
	Status    string `json:"status,omitempty"`
	Reason    string `json:"reason,omitempty"`
	At        int64  `json:"at,omitempty"`
}

type wireBroadcast struct {
	EventID    string            `json:"eventId"`
	Recipients []wireRecipient   `json:"recipients,omitempty"`
	Text       string            `json:"text,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type wireRecipient struct {
	ID    string `json:"id,omitempty"`
	Type  string `json:"type,omitempty"`
	Name  string `json:"name,omitempty"`
	Error string `json:"error,omitempty"`
}

type wireResponse struct {
	Success flexBool `json:"success"`
	Error   string   `json:"error,omitempty"`
}

type flexBool bool

func (b *flexBool) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if bytes.Equal(trimmed, []byte("null")) {
		return nil
	}

	if len(trimmed) > 1 && trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(trimmed, &s); err != nil {
			return err
		}

		parsed, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}

		*b = flexBool(parsed)

		return nil
	}

	var v bool
	if err := json.Unmarshal(trimmed, &v); err != nil {
		return err
	}

	*b = flexBool(v)

	return nil
}

func (b flexBool) MarshalJSON() ([]byte, error) {
	return json.Marshal(bool(b))
}
