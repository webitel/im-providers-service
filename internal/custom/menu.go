package custom

import (
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

// buildMenu renders an interactive message into the contract's menu shape. The
// button code is the internal button id, and it is what comes back in a
// callback, so a schema can match the press without depending on the label.
func buildMenu(in *sharedmodel.Interactive) *wireMenu {
	if in == nil {
		return nil
	}

	menu := &wireMenu{
		Text:      in.Body,
		SingleUse: in.SingleUse,
		Placement: placementOf(in.Placement),
	}

	switch {
	case in.Markup != nil:
		for _, row := range in.Markup.Rows {
			menu.Rows = append(menu.Rows, mapButtons(row.Buttons))
		}

	case in.ListReply != nil:
		if menu.Text == "" {
			menu.Text = in.ListReply.MainButtonTitle
		}

		for _, section := range in.ListReply.Sections {
			menu.Sections = append(menu.Sections, wireMenuGroup{
				Section: section.Section,
				Buttons: mapButtons(section.Buttons),
			})
		}
	}

	if len(menu.Rows) == 0 && len(menu.Sections) == 0 {
		return nil
	}

	return menu
}

func mapButtons(buttons []sharedmodel.KeyboardButton) []wireButton {
	out := make([]wireButton, 0, len(buttons))
	for _, b := range buttons {
		out = append(out, mapButton(b))
	}

	return out
}

func mapButton(b sharedmodel.KeyboardButton) wireButton {
	out := wireButton{Code: b.ID, Text: b.Label}

	switch {
	case b.URL != nil:
		out.URL = b.URL.URL
	case b.Callback != nil:
		out.Data = b.Callback.Data
	case b.Request != nil:
		out.Action = b.Request.Action
	}

	return out
}

func placementOf(p sharedmodel.MenuPlacement) string {
	switch p {
	case sharedmodel.MenuPlacementInline:
		return "inline"
	case sharedmodel.MenuPlacementPersistent:
		return "persistent"
	case sharedmodel.MenuPlacementUnspecified:
		return ""
	default:
		return ""
	}
}
