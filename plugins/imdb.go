// (c) Jisin0
// Functions and types to process imdb results.

package plugins

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/Jisin0/filmigo/imdb"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

var (
	imdbClient       = imdb.NewClient()
	searchMethodIMDb = "imdb"
)

const (
	imdbLogo     = "https://telegra.ph/file/1720930421ae2b00d9bab.jpg"
	imdbBanner   = "https://telegra.ph/file/2dd6f7c9ebfb237db4826.jpg"
	imdbHomepage = "https://imdb.com"
)

func IMDbInlineSearch(query string) []gotgbot.InlineQueryResult {
	results := OMDbInlineSearch(query)
	for i := range results {
		if photoResult, ok := results[i].(gotgbot.InlineQueryResultArticle); ok {
			photoResult.Id = strings.Replace(photoResult.Id, searchMethodOMDb, searchMethodIMDb, 1)
			if photoResult.ReplyMarkup != nil && len(photoResult.ReplyMarkup.InlineKeyboard) > 0 {
				callbackData := photoResult.ReplyMarkup.InlineKeyboard[0][0].CallbackData
				photoResult.ReplyMarkup.InlineKeyboard[0][0].CallbackData = strings.Replace(callbackData, searchMethodOMDb, searchMethodIMDb, 1)
			}
			results[i] = photoResult
		}
	}
	return results
}

// --- FIX: Updated signature to match GetOMDbTitle ---
func GetIMDbTitle(id string, progress func(string)) (string, string, [][]gotgbot.InlineKeyboardButton, error) {
	return GetOMDbTitle(id, progress)
}

func IMDbCommand(bot *gotgbot.Bot, ctx *ext.Context) error {
	update := ctx.EffectiveMessage

	split := strings.SplitN(update.GetText(), " ", 2)
	if len(split) < 2 {
		text := "<i>Please provide a search query or movie id along with this command !\nFor Example:</i>\n  <code>/imdb Inception</code>\n  <code>/imdb tt1375666</code>"
		update.Reply(bot, text, &gotgbot.SendMessageOpts{ParseMode: gotgbot.ParseModeHTML})
		return nil
	}

	input := split[1]

	var (
		messageText string
		buttons     [][]gotgbot.InlineKeyboardButton
		err         error
		previewURL  string = imdbBanner
	)

	if id := regexp.MustCompile(`tt\d+`).FindString(input); id != "" {
		// Pass nil for progress as we don't handle intermediate edits in command yet
		pURL, caption, btns, e := GetOMDbTitle(id, nil)
		if e != nil {
			err = e
		} else {
			previewURL = pURL
			messageText = fmt.Sprintf("<a href=\"%s\">&#8203;</a>%s", previewURL, caption)
			buttons = btns
		}
	} else {
		results, e := SearchOMDb(input)
		if e != nil {
			err = e
		} else {
			previewURL = imdbBanner
			messageText = fmt.Sprintf("<a href=\"%s\">&#8203;</a><i>👋 Hey <tg-spoiler>%s</tg-spoiler> I've got %d Results for you 👇</i>", previewURL, mention(ctx.EffectiveUser), len(results))
			for _, r := range results {
				buttons = append(buttons, []gotgbot.InlineKeyboardButton{{Text: fmt.Sprintf("%s (%d)", r.Title, r.Year), CallbackData: fmt.Sprintf("open_%s_%s", searchMethodIMDb, r.ID)}})
			}
		}
	}

	if err != nil {
		previewURL = imdbBanner
		messageText = fmt.Sprintf("<a href=\"%s\">&#8203;</a><i>I'm Sorry %s I Couldn't find Anything for <code>%s</code> 🤧</i>", previewURL, mention(ctx.EffectiveUser), input)
		buttons = [][]gotgbot.InlineKeyboardButton{{{Text: "Search On Google 🔎", Url: fmt.Sprintf("https://google.com/search?q=%s", url.QueryEscape(input))}}}
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, messageText, &gotgbot.SendMessageOpts{
		ParseMode:   gotgbot.ParseModeHTML,
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: buttons},
		LinkPreviewOptions: &gotgbot.LinkPreviewOptions{
			IsDisabled:    false,
			ShowAboveText: true,
			Url:           previewURL,
		},
	})
	if err != nil {
		fmt.Printf("imdbcommand: %v", err)
	}

	return ext.EndGroups
}
