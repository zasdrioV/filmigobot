// (c) Jisin0
// Handle the chosen_inline_result event.

package plugins

import (
	"fmt"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

func InlineResultHandler(bot *gotgbot.Bot, ctx *ext.Context) error {
	var (
		update = ctx.ChosenInlineResult
		data   = update.ResultId
	)

	if data == notAvailable {
		return nil
	}

	args := strings.Split(data, "_")
	if len(args) < 2 {
		fmt.Println("bad resultid on choseninlineresult : " + data)
		return nil
	}

	var (
		method = args[0]
		id     = args[1]
	)

	// --- FIX: Pass status updater ---
	statusUpdater := func(msg string) {
		bot.EditMessageText(msg, &gotgbot.EditMessageTextOpts{
			InlineMessageId: update.InlineMessageId,
			ParseMode:       gotgbot.ParseModeHTML,
		})
	}
	
	previewURL, caption, buttons, err := getChosenResult(method, id, statusUpdater)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	messageText := fmt.Sprintf("<a href=\"%s\">&#8203;</a>%s", previewURL, caption)

	_, _, err = bot.EditMessageText(
		messageText,
		&gotgbot.EditMessageTextOpts{
			InlineMessageId: update.InlineMessageId,
			ParseMode:       gotgbot.ParseModeHTML,
			ReplyMarkup:     gotgbot.InlineKeyboardMarkup{InlineKeyboard: buttons},
			LinkPreviewOptions: &gotgbot.LinkPreviewOptions{
				IsDisabled:      false,
				ShowAboveText: true,
				Url:             previewURL,
			},
		},
	)
	if err != nil {
		fmt.Println(err)
	}

	return nil
}

// --- FIX: Signature updated ---
func getChosenResult(method, id string, progress func(string)) (string, string, [][]gotgbot.InlineKeyboardButton, error) {
	switch method {
	case searchMethodIMDb:
		return GetOMDbTitle(id, progress)
	case searchMethodOMDb:
		return GetOMDbTitle(id, progress)
	default:
		fmt.Println("unknown method on choseninlineresult : " + method)
		return GetOMDbTitle(id, progress)
	}
}

func CbOpen(bot *gotgbot.Bot, ctx *ext.Context) error {
	update := ctx.CallbackQuery

	split := strings.Split(update.Data, "_")
	if len(split) < 3 {
		update.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "Bad Callback Data !", ShowAlert: true})
		return ext.EndGroups
	}

	var (
		method = split[1]
		id     = split[2]
		
		previewURL string
		caption   string
		buttons   [][]gotgbot.InlineKeyboardButton
		err       error
	)

	// --- FIX: Pass status updater ---
	statusUpdater := func(msg string) {
		update.Message.EditText(bot, msg, &gotgbot.EditMessageTextOpts{ParseMode: gotgbot.ParseModeHTML})
	}

	switch method {
	case searchMethodIMDb:
		previewURL, caption, buttons, err = GetOMDbTitle(id, statusUpdater)
	case searchMethodOMDb:
		previewURL, caption, buttons, err = GetOMDbTitle(id, statusUpdater)
	default:
		fmt.Println("unknown method on cbopen: " + method)
		previewURL, caption, buttons, err = GetOMDbTitle(id, statusUpdater)
	}

	if err != nil {
		fmt.Printf("cbopen: %v", err)
		update.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "I Couldn't Fetch Data on That Movie 🤧\nPlease Try Again Later or Contact Admins !", ShowAlert: true})
		return nil
	}

	messageText := fmt.Sprintf("<a href=\"%s\">&#8203;</a>%s", previewURL, caption)

	_, _, err = update.Message.EditText(bot, messageText, &gotgbot.EditMessageTextOpts{
		ParseMode:   gotgbot.ParseModeHTML,
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: buttons},
		LinkPreviewOptions: &gotgbot.LinkPreviewOptions{
			IsDisabled:      false,
			ShowAboveText: true,
			Url:             previewURL,
		},
	})
	if err != nil {
		fmt.Printf("cbopen: %v", err)
	}

	return nil
}
