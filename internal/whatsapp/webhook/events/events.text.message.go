package events

type TextMessageEvent struct {
	BaseMessageEvent `json:",inline"`

	Text string
}

func NewTextMessageEven(base BaseMessageEvent, text string) *TextMessageEvent {
	return &TextMessageEvent{BaseMessageEvent: base, Text: text}
}
