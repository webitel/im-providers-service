package viber

import (
	"strings"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

// richMedia is Viber's carousel content message: buttons rendered inside the message
// bubble, which is the closest equivalent of an inline menu. Unlike a keyboard it stays
// in the conversation history — and, since Viber cannot edit a sent message, it can
// never be redrawn in place.
// https://developers.viber.com/docs/api/rest-bot-api/#rich-media-message--carousel-content-message
type richMedia struct {
	Type                string   `json:"Type"`
	ButtonsGroupColumns int      `json:"ButtonsGroupColumns"`
	ButtonsGroupRows    int      `json:"ButtonsGroupRows"`
	BgColor             string   `json:"BgColor,omitempty"`
	Buttons             []button `json:"Buttons"`
}

const (
	richMediaMaxColumns      = 6
	richMediaMaxRows         = 7
	minAPIVersionRichMedia   = 2
	minAPIVersionHiddenInput = 4
)

func buildRichMedia(interactive *sharedmodel.Interactive) *richMedia {
	if interactive == nil {
		return nil
	}

	var buttons []button

	switch {
	case interactive.Markup != nil:
		for _, row := range interactive.Markup.Rows {
			for _, b := range row.Buttons {
				buttons = append(buttons, mapButton(b, richMediaMaxColumns))
			}
		}
	case interactive.ListReply != nil:
		for _, section := range interactive.ListReply.Sections {
			if section.Section != "" {
				buttons = append(buttons, button{
					Columns:    richMediaMaxColumns,
					Rows:       1,
					ActionType: "none",
					ActionBody: " ",
					Text:       section.Section,
				})
			}

			for _, b := range section.Buttons {
				buttons = append(buttons, mapButton(b, richMediaMaxColumns))
			}
		}
	}

	if len(buttons) == 0 {
		return nil
	}

	rows := len(buttons)
	if rows > richMediaMaxRows {
		rows = richMediaMaxRows
	}

	for i := range buttons {
		buttons[i].Columns = richMediaMaxColumns
		buttons[i].Rows = 1
	}

	return &richMedia{
		Type:                "rich_media",
		ButtonsGroupColumns: richMediaMaxColumns,
		ButtonsGroupRows:    rows,
		Buttons:             buttons,
	}
}

func altTextFor(body string, rm *richMedia) string {
	if rm == nil {
		return body
	}

	labels := make([]string, 0, len(rm.Buttons))
	for _, b := range rm.Buttons {
		if b.ActionType == "none" || b.Text == "" {
			continue
		}
		labels = append(labels, b.Text)
	}

	if len(labels) == 0 {
		return body
	}
	if body == "" {
		return strings.Join(labels, " / ")
	}

	return body + "\n" + strings.Join(labels, " / ")
}
