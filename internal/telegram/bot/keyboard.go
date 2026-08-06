package bot

import (
	coremodel "github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/im-providers-service/internal/telegram/bot/model"
)

func parseInlineKeyboard(in *coremodel.KeyboardMarkup) (*model.InlineKeyboardMarkup, error) {
	var keyboard model.InlineKeyboardMarkup

	for _, row := range in.Rows {
		var newKeyboardRow []model.InlineKeyboardButton
		for _, button := range row.Buttons {
			newKeyboardRow = append(newKeyboardRow, parseInlineButton(button))
		}
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, newKeyboardRow)
	}

	return &keyboard, nil
}

func parseInlineButton(button coremodel.KeyboardButton) model.InlineKeyboardButton {
	convertedButton := model.InlineKeyboardButton{
		Text: button.Label,
	}
	if button.Callback != nil {
		convertedButton.CallbackData = &button.Callback.Data
	}
	if button.URL != nil {
		convertedButton.URL = &button.URL.URL
	}

	return convertedButton
}

func parseReplyKeyboard(in *coremodel.KeyboardListReply, singleUse bool) (*model.ReplyKeyboardMarkup, error) {
	var keyboard model.ReplyKeyboardMarkup

	for _, row := range in.Sections {
		var newKeyboardRow []model.KeyboardButton
		for _, button := range row.Buttons {
			newKeyboardRow = append(newKeyboardRow, parseReplyButton(button))
		}
		keyboard.Keyboard = append(keyboard.Keyboard, newKeyboardRow)
	}
	if singleUse {
		keyboard.OneTimeKeyboard = new(true)
	}

	return &keyboard, nil
}

func parseReplyButton(button coremodel.KeyboardButton) model.KeyboardButton {
	convertedButton := model.KeyboardButton{
		Text: button.Label,
	}

	return convertedButton
}
