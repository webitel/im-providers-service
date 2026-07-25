package viber

import sharedmodel "github.com/webitel/im-providers-service/internal/core/model"

// keyboard is a Viber keyboard attached to a message.
// https://developers.viber.com/docs/api/rest-bot-api/#keyboards
type keyboard struct {
	Type          string   `json:"Type"`
	DefaultHeight bool     `json:"DefaultHeight,omitempty"`
	Buttons       []button `json:"Buttons"`
}

// button is a single Viber keyboard button. Viber lays buttons out on a flat
// 6-column grid; Columns/Rows control how much space each button occupies.
type button struct {
	Columns    int    `json:"Columns,omitempty"`
	Rows       int    `json:"Rows,omitempty"`
	ActionType string `json:"ActionType"`
	ActionBody string `json:"ActionBody"`
	Text       string `json:"Text,omitempty"`
}

const viberGridColumns = 6

// buildKeyboard maps the platform-agnostic Interactive payload to a Viber keyboard.
// Returns nil when there are no usable buttons.
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
				// Section title rendered as a non-interactive header spanning the full row.
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
	return &keyboard{Type: "keyboard", DefaultHeight: true, Buttons: buttons}
}

// mapRow converts a logical row of buttons, distributing the 6-column grid evenly.
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
	case b.Callback != nil:
		out.ActionType = "reply"
		out.ActionBody = b.Callback.Data
	case b.Request != nil:
		switch b.Request.Action {
		case "location":
			out.ActionType = "location-picker"
			out.ActionBody = "location"
		case "phone", "contact", "user_phone_number":
			out.ActionType = "share-phone"
			out.ActionBody = "phone"
		default:
			// No Viber equivalent (e.g. email) — degrade to a reply carrying the action.
			out.ActionType = "reply"
			out.ActionBody = b.Request.Action
		}
	default:
		out.ActionType = "reply"
		out.ActionBody = b.Label
	}
	if out.ActionBody == "" {
		out.ActionBody = b.Label
	}
	return out
}
