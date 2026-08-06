package viber

import (
	"strings"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

// keyboard is a Viber keyboard attached to a message. It is displayed over the input
// field and stays there until another keyboard replaces it — Viber has no way to remove
// one, and only one keyboard is ever active per conversation.
// https://developers.viber.com/docs/api/rest-bot-api/#keyboards
type keyboard struct {
	Type            string   `json:"Type"`
	DefaultHeight   bool     `json:"DefaultHeight,omitempty"`
	Buttons         []button `json:"Buttons"`
	InputFieldState string   `json:"InputFieldState,omitempty"`
}

type button struct {
	Columns    int    `json:"Columns,omitempty"`
	Rows       int    `json:"Rows,omitempty"`
	ActionType string `json:"ActionType"`
	ActionBody string `json:"ActionBody"`
	Text       string `json:"Text,omitempty"`
	Silent     bool   `json:"Silent,omitempty"`
}

const viberGridColumns = 6

const (
	inputFieldRegular   = "regular"
	inputFieldMinimized = "minimized"
	inputFieldHidden    = "hidden"
)

func buildKeyboard(interactive *sharedmodel.Interactive) *keyboard {
	if interactive == nil {
		return nil
	}

	var buttons []button
	switch {
	case interactive.Markup != nil:
		for _, row := range interactive.Markup.Rows {
			buttons = append(buttons, mapRow(row.Buttons)...)
		}
	case interactive.ListReply != nil:
		for _, section := range interactive.ListReply.Sections {
			if section.Section != "" {
				buttons = append(buttons, button{
					Columns:    viberGridColumns,
					ActionType: "none",
					ActionBody: " ",
					Text:       section.Section,
				})
			}
			buttons = append(buttons, mapRow(section.Buttons)...)
		}
	}

	if len(buttons) == 0 {
		return nil
	}

	return &keyboard{
		Type:            "keyboard",
		DefaultHeight:   true,
		Buttons:         buttons,
		InputFieldState: inputFieldStateOf(interactive.InputFieldState),
	}
}

func inputFieldStateOf(state sharedmodel.InputFieldState) string {
	switch state {
	case sharedmodel.InputFieldStateRegular:
		return inputFieldRegular
	case sharedmodel.InputFieldStateMinimized:
		return inputFieldMinimized
	case sharedmodel.InputFieldStateHidden:
		return inputFieldHidden
	case sharedmodel.InputFieldStateUnspecified:
		return ""
	default:
		return ""
	}
}

func interactiveButtons(interactive *sharedmodel.Interactive) []sharedmodel.KeyboardButton {
	if interactive == nil {
		return nil
	}

	var out []sharedmodel.KeyboardButton

	switch {
	case interactive.Markup != nil:
		for _, row := range interactive.Markup.Rows {
			out = append(out, row.Buttons...)
		}
	case interactive.ListReply != nil:
		for _, section := range interactive.ListReply.Sections {
			out = append(out, section.Buttons...)
		}
	}

	return out
}

func mapRow(src []sharedmodel.KeyboardButton) []button {
	if len(src) == 0 {
		return nil
	}
	cols := viberGridColumns / len(src)
	if cols < 1 {
		cols = 1
	}
	out := make([]button, 0, len(src))
	for _, b := range src {
		out = append(out, mapButton(b, cols))
	}
	return out
}

func mapButton(b sharedmodel.KeyboardButton, cols int) button {
	out := button{Columns: cols, Text: b.Label}

	switch {
	case b.URL != nil:
		out.ActionType = "open-url"
		out.ActionBody = b.URL.URL
	case b.Request != nil && viberRequestAction(b.Request.Action) != "":
		out.ActionType = viberRequestAction(b.Request.Action)
		out.ActionBody = strings.TrimPrefix(out.ActionType, "share-")
	default:
		out.ActionType = "reply"
		out.ActionBody, _ = replyPayload(b)
		out.Silent = true
	}

	if out.ActionBody == "" {
		out.ActionBody = b.Label
	}

	return out
}

func viberRequestAction(action string) string {
	switch action {
	case "location":
		return "location-picker"
	case "phone", "contact", "user_phone_number":
		return "share-phone"
	default:
		return ""
	}
}

func replyPayload(b sharedmodel.KeyboardButton) (string, bool) {
	switch {
	case b.URL != nil:
		return "", false
	case b.Callback != nil && b.Callback.Data != "":
		return b.Callback.Data, true
	case b.Request != nil && viberRequestAction(b.Request.Action) != "":
		return "", false
	case b.Request != nil:
		return b.Request.Action, true
	default:
		return b.Label, true
	}
}
