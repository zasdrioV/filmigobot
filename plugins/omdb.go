// (c) Jisin0
// Functions and types to search using the unofficial IMDb API.

package plugins

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Jisin0/filmigo/omdb"
	"github.com/PaulSonOfLars/gotgbot/v2"
)

const (
	omdbBanner   = "https://telegra.ph/file/e810982a269773daa42a9.png"
	omdbHomepage = "https://imdb.com"
	notAvailable = "N/A"
	
	unofficialAPI = "https://imdb.iamidiotareyoutoo.com/search"

	// --- CONFIGURATION ---
	topCastLimit    = 30
	enableAIReview  = true  
	enableTelegraph = false  
)

var (
	omdbClient       *omdb.OmdbClient
	searchMethodOMDb = "omdb"
	telegraphToken   string
)

func init() {
	if OmdbApiKey != "" {
		omdbClient = omdb.NewClient(OmdbApiKey)
		inlineSearchButtons = append(inlineSearchButtons, []gotgbot.InlineKeyboardButton{{Text: "🔍 Search OMDb", SwitchInlineQueryCurrentChat: &inlineOMDbSwitch}})
	}
	if enableTelegraph {
		go ensureTelegraphToken()
	}
}

// --- TELEGRAPH HELPER FUNCTIONS ---

func ensureTelegraphToken() {
	if telegraphToken != "" { return }
	apiURL := "https://api.telegra.ph/createAccount?short_name=FilmigoBot&author_name=Filmigo+Bot"
	resp, err := http.Get(apiURL)
	if err == nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var res struct {
			Ok     bool `json:"ok"`
			Result struct { AccessToken string `json:"access_token"` } `json:"result"`
		}
		json.Unmarshal(body, &res)
		if res.Ok { telegraphToken = res.Result.AccessToken }
	}
}

type tgNode struct {
	Tag      string   `json:"tag"`
	Attrs    *tgAttrs `json:"attrs,omitempty"`
	Children []any    `json:"children,omitempty"`
}
type tgAttrs struct {
	Src  string `json:"src,omitempty"`
	Href string `json:"href,omitempty"`
}

func createTelegraphPage(title string, nodes []tgNode) string {
	ensureTelegraphToken()
	if telegraphToken == "" { return "" }
	
	contentBytes, err := json.Marshal(nodes)
	if err != nil { return "" }

	data := url.Values{}
	data.Set("access_token", telegraphToken)
	data.Set("title", title)
	data.Set("content", string(contentBytes))
	data.Set("return_content", "false")

	resp, err := http.PostForm("https://api.telegra.ph/createPage", data)
	if err != nil { return "" }
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var res struct {
		Ok     bool `json:"ok"`
		Result struct { Url string `json:"url"` } `json:"result"`
	}
	json.Unmarshal(body, &res)
	return res.Result.Url
}

func makeRow(label, value string) tgNode {
	return tgNode{Tag: "p", Children: []any{ tgNode{Tag: "b", Children: []any{label + ": "}}, value }}
}
func makeHeader(text string) tgNode {
	return tgNode{Tag: "h4", Children: []any{text}}
}
func makeSubHeader(text string) tgNode {
	return tgNode{Tag: "h5", Children: []any{text}}
}

// --- STRUCTS FOR UNOFFICIAL API ---

type unofficialSearchResponse struct {
	Ok          bool `json:"ok"`
	Description []struct {
		ImdbID string `json:"#IMDB_ID"`
		Title  string `json:"#TITLE"`
		Year   int    `json:"#YEAR"`
		Poster string `json:"#IMG_POSTER"`
		Actors string `json:"#ACTORS"`
	} `json:"description"`
}

type unofficialBaseResponse struct {
	Ok bool `json:"ok"`
}

type unofficialDetailData struct {
	Ok    bool `json:"ok"`
	Short struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Trailer     struct { EmbedURL string `json:"embedUrl"` } `json:"trailer"`
	} `json:"short"`
	
	// Root Level
	ReviewSummary *struct { Overall struct { Medium struct { Value struct { PlaidHtml string `json:"plaidHtml"` } `json:"value"` } `json:"medium"` } `json:"overall"` } `json:"reviewSummary"`
	Keywords *struct { Edges []struct { Node struct { Text string `json:"text"` } `json:"node"` } `json:"edges"` } `json:"keywords"`
	MoreLikeThisTitles *struct { Edges []struct { Node struct { Id string `json:"id"`; TitleText struct { Text string `json:"text"` } `json:"titleText"` } `json:"node"` } `json:"edges"` } `json:"moreLikeThisTitles"`

	Top struct {
		TitleText struct { Text string `json:"text"` } `json:"titleText"`
		TitleType struct { Text string `json:"text"` } `json:"titleType"`
		ReleaseYear struct { Year int `json:"year"`; EndYear int `json:"endYear"` } `json:"releaseYear"`
		ReleaseDate struct {
			Day int `json:"day"`; Month int `json:"month"`; Year int `json:"year"`
			Country struct { Text string `json:"text"` } `json:"country"`
		} `json:"releaseDate"`
		Runtime struct {
			DisplayableProperty struct { Value struct { PlainText string `json:"plainText"` } `json:"value"` } `json:"displayableProperty"`
		} `json:"runtime"`
		RatingsSummary struct { AggregateRating float64 `json:"aggregateRating"`; VoteCount int `json:"voteCount"` } `json:"ratingsSummary"`
		Metacritic *struct { Metascore struct { Score int `json:"score"` } `json:"metascore"` } `json:"metacritic"`
		Genres struct { Genres []struct { Text string `json:"text"` } `json:"genres"` } `json:"genres"`
		Interests struct { Edges []struct { Node struct { PrimaryText struct { Text string `json:"text"` } `json:"primaryText"` } `json:"node"` } `json:"edges"` } `json:"interests"`
		Plot struct { PlotText struct { PlainText string `json:"plainText"` } `json:"plotText"` } `json:"plot"`
		PrimaryImage struct { URL string `json:"url"` } `json:"primaryImage"`
		Directors []struct { Credits []struct { Name struct { NameText struct { Text string `json:"text"` } `json:"nameText"`; ID string `json:"id"` } `json:"name"` } `json:"credits"` } `json:"directorsPageTitle"`
		PrincipalCredits []struct { Grouping struct { Text string `json:"text"` } `json:"grouping"`; Credits []struct { Name struct { NameText struct { Text string `json:"text"` } `json:"nameText"`; ID string `json:"id"` } `json:"name"` } `json:"credits"` } `json:"principalCreditsV2"`
		Cast []struct { Grouping struct { Text string `json:"text"` } `json:"grouping"`; Credits []struct { Name struct { NameText struct { Text string `json:"text"` } `json:"nameText"`; ID string `json:"id"` } `json:"name"` } `json:"credits"` } `json:"castV2"`
		Certificate struct { Rating string `json:"rating"` } `json:"certificate"`
		ProductionStatus struct { CurrentProductionStage struct { Text string `json:"text"` } `json:"currentProductionStage"` } `json:"productionStatus"`
		FeaturedReviews *struct { Edges []struct { Node struct { Author struct { NickName string `json:"nickName"` } `json:"author"`; Summary struct { OriginalText string `json:"originalText"` } `json:"summary"`; Text struct { OriginalText struct { PlainHtml string `json:"plaidHtml"` } `json:"originalText"` } `json:"text"`; AuthorRating int `json:"authorRating"` } `json:"node"` } `json:"edges"` } `json:"featuredReviews"`
	} `json:"top"`
	
	Main struct {
		PrestigiousAwardSummary *struct { Nominations int `json:"nominations"`; Wins int `json:"wins"` } `json:"prestigiousAwardSummary"`
		Wins struct { Total int `json:"total"` } `json:"wins"`
		Nominations struct { Total int `json:"total"` } `json:"nominationsExcludeWins"`
		Languages struct { Languages []struct { Text string `json:"text"` } `json:"spokenLanguages"` } `json:"spokenLanguages"`
		Countries struct { Countries []struct { Text string `json:"text"` } `json:"countries"` } `json:"countriesDetails"`
		Akas struct { Edges []struct { Node struct { Text string `json:"text"` } `json:"node"` } `json:"edges"` } `json:"akas"`
		Cast []struct { Grouping struct { Text string `json:"text"` } `json:"grouping"`; Credits []struct { Name struct { NameText struct { Text string `json:"text"` } `json:"nameText"`; ID string `json:"id"` } `json:"name"`; Characters []struct { Name string `json:"name"` } `json:"characters"` } `json:"credits"` } `json:"castV2"`
		Episodes *struct { Seasons []struct { Number int `json:"number"` } `json:"seasons"`; TotalEpisodes struct { Total int `json:"total"` } `json:"totalEpisodes"` } `json:"episodes"`
		ProductionBudget *struct { Budget struct { Amount int64 `json:"amount"`; Currency string `json:"currency"` } `json:"budget"` } `json:"productionBudget"`
		LifetimeGross *struct { Total struct { Amount int64 `json:"amount"`; Currency string `json:"currency"` } `json:"total"` } `json:"lifetimeGross"`
		WorldwideGross *struct { Total struct { Amount int64 `json:"amount"`; Currency string `json:"currency"` } `json:"total"` } `json:"worldwideGross"`
		TechnicalSpecifications *struct {
			SoundMixes struct { Items []struct { Text string `json:"text"` } `json:"items"` } `json:"soundMixes"`
			AspectRatios struct { Items []struct { AspectRatio string `json:"aspectRatio"` } `json:"items"` } `json:"aspectRatios"`
		} `json:"technicalSpecifications"`
		TriviaTotal struct { Total int `json:"total"` } `json:"triviaTotal"`
		Trivia struct { Edges []struct { Node struct { Text struct { PlaidHtml string `json:"plaidHtml"` } `json:"text"` } `json:"node"` } `json:"edges"` } `json:"trivia"`
		GoofsTotal struct { Total int `json:"total"` } `json:"goofsTotal"`
		Goofs struct { Edges []struct { Node struct { Text struct { PlaidHtml string `json:"plaidHtml"` } `json:"text"` } `json:"node"` } `json:"edges"` } `json:"goofs"`
		QuotesTotal struct { Total int `json:"total"` } `json:"quotesTotal"`
		Quotes struct { Edges []struct { Node struct { Lines []struct { Text string `json:"text"` } `json:"lines"` } `json:"node"` } `json:"edges"` } `json:"quotes"`
		FilmingLocations struct { Edges []struct { Node struct { Text string `json:"text"` } `json:"node"` } `json:"edges"` } `json:"filmingLocations"`
		Production struct { Edges []struct { Node struct { Company struct { CompanyText struct { Text string `json:"text"` } `json:"companyText"` } `json:"company"` } `json:"node"` } `json:"edges"` } `json:"production"`
		Soundtrack struct { Edges []struct { Node struct { Text string `json:"text"` } `json:"node"` } `json:"edges"` } `json:"soundtrack"`
	} `json:"main"`
}

func OMDbInlineSearch(query string) []gotgbot.InlineQueryResult {
	apiURL := fmt.Sprintf("%s?q=%s", unofficialAPI, url.QueryEscape(query))
	resp, err := http.Get(apiURL)
	if err != nil { return nil }
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil { return nil }

	var searchData unofficialSearchResponse
	if err := json.Unmarshal(body, &searchData); err != nil || !searchData.Ok { return nil }

	results := make([]gotgbot.InlineQueryResult, 0, len(searchData.Description))
	for _, item := range searchData.Description {
		posterURL := item.Poster
		if posterURL == "" { posterURL = omdbBanner }
		title := fmt.Sprintf("%s (%d)", item.Title, item.Year)
		results = append(results, gotgbot.InlineQueryResultArticle{
			Id:           searchMethodOMDb + "_" + item.ImdbID,
			Title:        title,
			Description:  item.Actors,
			ThumbnailUrl: posterURL,
			InputMessageContent: gotgbot.InputTextMessageContent{
				MessageText: fmt.Sprintf("<i>Loading details for %s...</i>", title),
				ParseMode:   gotgbot.ParseModeHTML,
			},
			ReplyMarkup: &gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
				{{Text: "Open IMDb", CallbackData: fmt.Sprintf("open_%s_%s", searchMethodOMDb, item.ImdbID)}},
			}},
		})
	}
	return results
}

func formatDuration(runtime string) string {
	runtime = strings.TrimSpace(strings.Replace(runtime, "min", "", 1))
	totalMinutes, err := strconv.Atoi(runtime)
	if err != nil { return fmt.Sprintf("%s min", runtime) }
	if totalMinutes < 60 { return fmt.Sprintf("%d min", totalMinutes) }
	hours := totalMinutes / 60
	minutes := totalMinutes % 60
	if minutes == 0 { return fmt.Sprintf("%dh", hours) }
	return fmt.Sprintf("%dh %dmin", hours, minutes)
}

func GetOMDbTitle(id string) (string, string, [][]gotgbot.InlineKeyboardButton, error) {
	var buttons [][]gotgbot.InlineKeyboardButton
	apiURL := fmt.Sprintf("%s?tt=%s", unofficialAPI, id)
	resp, err := http.Get(apiURL)
	if err != nil { return "", "", buttons, fmt.Errorf("failed to call API: %w", err) }
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil { return "", "", buttons, fmt.Errorf("failed to read API response: %w", err) }

	var baseCheck unofficialBaseResponse
	if err := json.Unmarshal(body, &baseCheck); err == nil && !baseCheck.Ok {
		return "", "", buttons, errors.New("API returned an error (ok:false), movie likely not found")
	}
	var title unofficialDetailData
	if err := json.Unmarshal(body, &title); err != nil {
		return "", "", buttons, fmt.Errorf("failed to parse API JSON: %w", err)
	}
	if title.Top.TitleText.Text == "" {
		return "", "", buttons, errors.New("movie not found or API failed")
	}

	isSeries := (title.Top.TitleType.Text == "TV Series" || title.Top.TitleType.Text == "TV Mini Series")
	genreEmojiMap := map[string]string{
		"Action": "💥", "Adventure": "🗺️", "Sci-Fi": "🚀", "Comedy": "🤣", "Drama": "🎭", "Romance": "🌹",
		"Thriller": "🔪", "Horror": "👻", "Fantasy": "✨",
		"Mystery": "❓", "Crime": "-", "Animation": "-",
		"War": "-", "History": "-","Music": "🎶",
	}
	countryFlagMap := map[string]string{
		"United States": "🇺🇸", "USA": "🇺🇸", "United Kingdom": "🇬🇧", "UK": "🇬🇧", "India": "🇮🇳", "France": "🇫🇷", "Japan": "🇯🇵", "Canada": "🇨🇦", "Germany": "🇩🇪",
	}
	monthMap := map[int]string{
		1: "January", 2: "February", 3: "March", 4: "April", 5: "May", 6: "June", 7: "July", 8: "August", 9: "September", 10: "October", 11: "November", 12: "December",
	}
	genreMap := make(map[string]bool)
	var captionBuilder strings.Builder
	imdbURL := omdbHomepage + "/title/" + id

	// --- CAPTION GENERATION ---
	var yearString string
	if isSeries && title.Top.ReleaseYear.EndYear > 0 {
		yearString = fmt.Sprintf("[%d-%d]", title.Top.ReleaseYear.Year, title.Top.ReleaseYear.EndYear)
	} else if isSeries && title.Top.ReleaseYear.EndYear == 0 {
		yearString = fmt.Sprintf("[%d-Present]", title.Top.ReleaseYear.Year)
	} else {
		yearString = fmt.Sprintf("[%d]", title.Top.ReleaseYear.Year)
	}
	captionBuilder.WriteString(fmt.Sprintf("<i>%s: </i><b>%s %s</b> | <a href=\"%s\">IMDb Link</a>\n", title.Top.TitleType.Text, title.Top.TitleText.Text, yearString, imdbURL))

	if len(title.Main.Akas.Edges) > 0 {
		captionBuilder.WriteString(fmt.Sprintf("<i>(AKA: %s)</i>\n", title.Main.Akas.Edges[0].Node.Text))
	}

	if isSeries && title.Main.Episodes != nil {
		seasonCount := len(title.Main.Episodes.Seasons)
		episodeCount := title.Main.Episodes.TotalEpisodes.Total
		if seasonCount > 0 && episodeCount > 0 {
			captionBuilder.WriteString(fmt.Sprintf("<b>%d Seasons (%d Episodes)</b>\n", seasonCount, episodeCount))
		}
	}

	if title.Top.Runtime.DisplayableProperty.Value.PlainText != "" {
		durationStr := title.Top.Runtime.DisplayableProperty.Value.PlainText
		if isSeries { durationStr += "/Episode" }
		captionBuilder.WriteString(fmt.Sprintf("<i>Duration: </i>%s\n", durationStr))
	}
	
	rd := title.Top.ReleaseDate
	if rd.Year > 0 && rd.Month > 0 && rd.Day > 0 {
		if monthName, ok := monthMap[rd.Month]; ok {
			country := rd.Country.Text
			dateStr := fmt.Sprintf("<i>Release Date: </i>%d %s %d (%s)", rd.Day, monthName, rd.Year, country)
			if isSeries { dateStr += " - For First Episode" }
			captionBuilder.WriteString(dateStr + "\n")
		}
	}

	ratingStr := ""
	if title.Top.RatingsSummary.AggregateRating > 0 {
		ratingStr = fmt.Sprintf("<i>Rating ⭐️ </i><b>%.1f / 10</b> (from %d votes)", title.Top.RatingsSummary.AggregateRating, title.Top.RatingsSummary.VoteCount)
	}
	if title.Top.Metacritic != nil && title.Top.Metacritic.Metascore.Score > 0 {
		if ratingStr != "" { ratingStr += " | " }
		ratingStr += fmt.Sprintf("<b>Ⓜ️ %d/100</b>", title.Top.Metacritic.Metascore.Score)
	}
	if ratingStr != "" { captionBuilder.WriteString(ratingStr + "\n") }

	captionBuilder.WriteString("<blockquote>")
	if len(title.Top.Genres.Genres) > 0 {
		var formattedGenres []string
		for _, g := range title.Top.Genres.Genres {
			emoji := "- "
			if e, ok := genreEmojiMap[g.Text]; ok { emoji = e + " " }
			formattedGenres = append(formattedGenres, fmt.Sprintf("%s#%s", emoji, g.Text))
			genreMap[g.Text] = true
		}
		captionBuilder.WriteString(fmt.Sprintf("<i>Genres: </i>%s\n", strings.Join(formattedGenres, " ")))
	}
	if len(title.Top.Interests.Edges) > 0 {
		var formattedThemes []string
		for _, t := range title.Top.Interests.Edges {
			themeName := t.Node.PrimaryText.Text
			if _, isGenre := genreMap[themeName]; !isGenre {
				themeTag := strings.ReplaceAll(themeName, " ", "_")
				formattedThemes = append(formattedThemes, fmt.Sprintf("#%s", themeTag))
			}
		}
		if len(formattedThemes) > 0 {
			captionBuilder.WriteString(fmt.Sprintf("<i>Themes: </i>%s\n", strings.Join(formattedThemes, " ")))
		}
	}
	var formattedLangs []string
	for _, l := range title.Main.Languages.Languages { formattedLangs = append(formattedLangs, "#"+l.Text) }
	var formattedCountries []string
	for _, c := range title.Main.Countries.Countries {
		flag := ""
		if f, ok := countryFlagMap[c.Text]; ok { flag = f + " " }
		countryTag := strings.ReplaceAll(c.Text, " ", "_")
		formattedCountries = append(formattedCountries, fmt.Sprintf("%s#%s", flag, countryTag))
	}
	captionBuilder.WriteString(fmt.Sprintf("<i>Language (Country): </i>%s (%s)", strings.Join(formattedLangs, " "), strings.Join(formattedCountries, " ")))
	captionBuilder.WriteString("</blockquote>\n\n")

	if title.Top.Plot.PlotText.PlainText != "" {
		captionBuilder.WriteString(fmt.Sprintf("<blockquote><b>Story Line: </b><i>%s</i></blockquote>\n\n", title.Top.Plot.PlotText.PlainText))
	}

	if enableAIReview && title.ReviewSummary != nil && title.ReviewSummary.Overall.Medium.Value.PlaidHtml != "" {
		reviewText := html.UnescapeString(title.ReviewSummary.Overall.Medium.Value.PlaidHtml)
		captionBuilder.WriteString(fmt.Sprintf("<blockquote><b>AI Review: </b><i>%s</i></blockquote>\n\n", reviewText))
	}

	captionBuilder.WriteString("<blockquote>")
	var directors []string
	if len(title.Top.Directors) > 0 && len(title.Top.Directors[0].Credits) > 0 {
		for _, d := range title.Top.Directors[0].Credits { directors = append(directors, fmt.Sprintf("<a href=\"%s/name/%s\">%s</a>", omdbHomepage, d.Name.ID, d.Name.NameText.Text)) }
	}
	if len(directors) == 0 {
		for _, group := range title.Top.PrincipalCredits {
			if group.Grouping.Text == "Director" || group.Grouping.Text == "Directors" {
				for _, d := range group.Credits { directors = append(directors, fmt.Sprintf("<a href=\"%s/name/%s\">%s</a>", omdbHomepage, d.Name.ID, d.Name.NameText.Text)) }
			}
		}
	}
	var creators []string
	if isSeries {
		for _, group := range title.Top.PrincipalCredits {
			if group.Grouping.Text == "Creator" || group.Grouping.Text == "Creators" {
				for _, c := range group.Credits { creators = append(creators, fmt.Sprintf("<a href=\"%s/name/%s\">%s</a>", omdbHomepage, c.Name.ID, c.Name.NameText.Text)) }
			}
		}
	}
	if len(directors) > 0 { captionBuilder.WriteString(fmt.Sprintf("<i><b>Directors:</b></i> %s\n", strings.Join(directors, ", "))) } else if len(creators) > 0 { captionBuilder.WriteString(fmt.Sprintf("<i><b>Directors:</b></i> %s\n", strings.Join(creators, ", "))) }
	
	var writers []string
	var stars []string
	isStar := make(map[string]bool)
	for _, group := range title.Top.PrincipalCredits {
		if group.Grouping.Text == "Writers" || group.Grouping.Text == "Writer" {
			for _, w := range group.Credits { writers = append(writers, fmt.Sprintf("<a href=\"%s/name/%s\">%s</a>", omdbHomepage, w.Name.ID, w.Name.NameText.Text)) }
		}
		if group.Grouping.Text == "Stars" {
			for _, s := range group.Credits {
				stars = append(stars, fmt.Sprintf("<a href=\"%s/name/%s\">%s</a>", omdbHomepage, s.Name.ID, s.Name.NameText.Text))
				isStar[s.Name.NameText.Text] = true
			}
		}
	}
	if len(writers) > 0 { captionBuilder.WriteString(fmt.Sprintf("<i><b>Writers:</b></i> %s\n", strings.Join(writers, ", "))) }
	if len(stars) > 0 { captionBuilder.WriteString(fmt.Sprintf("<i><b>Stars:</b></i> %s\n", strings.Join(stars, ", "))) }

	var topCast []string
	for _, group := range title.Main.Cast {
		if group.Grouping.Text == "Top Cast" {
			for _, c := range group.Credits {
				if _, alreadyStar := isStar[c.Name.NameText.Text]; !alreadyStar {
					 if len(topCast) < topCastLimit { topCast = append(topCast, fmt.Sprintf("<a href=\"%s/name/%s\">%s</a>", omdbHomepage, c.Name.ID, c.Name.NameText.Text)) } else { break }
				}
			}
			break
		}
	}
	if len(topCast) > 0 { captionBuilder.WriteString(fmt.Sprintf("<i><b>Top Cast:</b></i> %s", strings.Join(topCast, ", "))) }
	captionBuilder.WriteString("</blockquote>\n\n")

	captionBuilder.WriteString("<blockquote>")
	awardsURL := fmt.Sprintf("%s/title/%s/awards", omdbHomepage, id)
	var awardsText string
	if title.Main.PrestigiousAwardSummary != nil {
		awardsText = fmt.Sprintf("Won %d Oscars. %d wins & %d nominations total.", title.Main.PrestigiousAwardSummary.Wins, title.Main.Wins.Total, title.Main.Nominations.Total)
	} else if title.Main.Wins.Total > 0 {
		awardsText = fmt.Sprintf("%d wins & %d nominations total.", title.Main.Wins.Total, title.Main.Nominations.Total)
	}
	if awardsText != "" { captionBuilder.WriteString(fmt.Sprintf("<b>Awards: </b><a href=\"%s\">%s</a>\n", awardsURL, awardsText)) }
	ottURL := fmt.Sprintf("https://www.justwatch.com/in/search?q=%s", url.QueryEscape(title.Top.TitleText.Text))
	captionBuilder.WriteString(fmt.Sprintf("<b>OTT Info: </b><a href=\"%s\">Find on JustWatch</a>", ottURL))
	captionBuilder.WriteString("</blockquote>")

	// --- TELEGRA.PH GENERATION (FIXED & EXPANDED) ---
	if enableTelegraph {
		var tNodes []tgNode
		
		tNodes = append(tNodes, tgNode{Tag: "h3", Children: []any{fmt.Sprintf("%s (%d)", title.Top.TitleText.Text, title.Top.ReleaseYear.Year)}})
		if title.Top.PrimaryImage.URL != "" {
			tNodes = append(tNodes, tgNode{Tag: "figure", Children: []any{ tgNode{Tag: "img", Attrs: &tgAttrs{Src: title.Top.PrimaryImage.URL}} }})
		}
		
		tNodes = append(tNodes, makeHeader("Info"))
		tNodes = append(tNodes, makeRow("Type", title.Top.TitleType.Text))
		tNodes = append(tNodes, makeRow("Rating", fmt.Sprintf("%.1f/10 (%d votes)", title.Top.RatingsSummary.AggregateRating, title.Top.RatingsSummary.VoteCount)))
		if title.Top.Certificate.Rating != "" { tNodes = append(tNodes, makeRow("Content Rating", title.Top.Certificate.Rating)) }
		tNodes = append(tNodes, makeRow("Status", title.Top.ProductionStatus.CurrentProductionStage.Text))
		
		// Fix: Correctly formatted Directors/Writers in Telegraph
		var tgDirectors []string
		if len(title.Top.Directors) > 0 && len(title.Top.Directors[0].Credits) > 0 {
			for _, d := range title.Top.Directors[0].Credits { tgDirectors = append(tgDirectors, d.Name.NameText.Text) }
		}
		if len(tgDirectors) > 0 { tNodes = append(tNodes, makeRow("Directors", strings.Join(tgDirectors, ", "))) }

		var tgWriters []string
		for _, group := range title.Top.PrincipalCredits {
			if group.Grouping.Text == "Writers" || group.Grouping.Text == "Writer" {
				for _, w := range group.Credits { tgWriters = append(tgWriters, w.Name.NameText.Text) }
			}
		}
		if len(tgWriters) > 0 { tNodes = append(tNodes, makeRow("Writers", strings.Join(tgWriters, ", "))) }
		
		// Keywords
		if title.Keywords != nil && len(title.Keywords.Edges) > 0 {
			var kws []string
			for _, k := range title.Keywords.Edges { kws = append(kws, k.Node.Text) }
			tNodes = append(tNodes, makeRow("Keywords", strings.Join(kws, ", ")))
		}

		if title.Top.Plot.PlotText.PlainText != "" {
			tNodes = append(tNodes, makeHeader("Plot"))
			tNodes = append(tNodes, tgNode{Tag: "p", Children: []any{title.Top.Plot.PlotText.PlainText}})
		}
		
		// AI Review in Telegraph
		if title.ReviewSummary != nil && title.ReviewSummary.Overall.Medium.Value.PlaidHtml != "" {
			tNodes = append(tNodes, makeHeader("AI Review Summary"))
			reviewText := html.UnescapeString(title.ReviewSummary.Overall.Medium.Value.PlaidHtml)
			tNodes = append(tNodes, tgNode{Tag: "i", Children: []any{reviewText}})
		}

		// Full Cast (Iterating ALL groups)
		if len(title.Main.Cast) > 0 {
			tNodes = append(tNodes, makeHeader("Full Cast & Crew"))
			for _, group := range title.Main.Cast {
				tNodes = append(tNodes, makeSubHeader(group.Grouping.Text))
				var members []string
				count := 0
				for _, c := range group.Credits {
					if count > 100 { break } // High limit
					charName := ""
					if len(c.Characters) > 0 { charName = " as " + c.Characters[0].Name }
					members = append(members, fmt.Sprintf("%s%s", c.Name.NameText.Text, charName))
					count++
				}
				tNodes = append(tNodes, tgNode{Tag: "p", Children: []any{strings.Join(members, ", ")}})
			}
		}

		// Reviews
		if title.Top.FeaturedReviews != nil && len(title.Top.FeaturedReviews.Edges) > 0 {
			tNodes = append(tNodes, makeHeader("Featured Reviews"))
			for _, r := range title.Top.FeaturedReviews.Edges {
				reviewContent := strings.ReplaceAll(html.UnescapeString(r.Node.Text.OriginalText.PlainHtml), "<br/>", "\n")
				tNodes = append(tNodes, tgNode{Tag: "blockquote", Children: []any{
					tgNode{Tag: "b", Children: []any{fmt.Sprintf("%s (%d/10): ", r.Node.Summary.OriginalText, r.Node.AuthorRating)}},
					reviewContent,
					tgNode{Tag: "i", Children: []any{" - by " + r.Node.Author.NickName}},
				}})
			}
		}

		// Tech Specs
		tNodes = append(tNodes, makeHeader("Technical Specs & Box Office"))
		if title.Main.ProductionBudget != nil { tNodes = append(tNodes, makeRow("Budget", fmt.Sprintf("%d %s", title.Main.ProductionBudget.Budget.Amount, title.Main.ProductionBudget.Budget.Currency))) }
		if title.Main.LifetimeGross != nil { tNodes = append(tNodes, makeRow("Lifetime Gross", fmt.Sprintf("%d %s", title.Main.LifetimeGross.Total.Amount, title.Main.LifetimeGross.Total.Currency))) }
		if title.Main.WorldwideGross != nil { tNodes = append(tNodes, makeRow("Worldwide Gross", fmt.Sprintf("%d %s", title.Main.WorldwideGross.Total.Amount, title.Main.WorldwideGross.Total.Currency))) }
		if title.Main.TechnicalSpecifications != nil {
			if len(title.Main.TechnicalSpecifications.AspectRatios.Items) > 0 { tNodes = append(tNodes, makeRow("Aspect Ratio", title.Main.TechnicalSpecifications.AspectRatios.Items[0].AspectRatio)) }
			if len(title.Main.TechnicalSpecifications.SoundMixes.Items) > 0 {
				var sounds []string
				for _, s := range title.Main.TechnicalSpecifications.SoundMixes.Items { sounds = append(sounds, s.Text) }
				tNodes = append(tNodes, makeRow("Sound Mix", strings.Join(sounds, ", ")))
			}
		}

		// Trivia, Goofs, Quotes (Increased Limits)
		if title.Main.TriviaTotal.Total > 0 {
			tNodes = append(tNodes, makeHeader(fmt.Sprintf("Trivia (%d items)", title.Main.TriviaTotal.Total)))
			count := 0
			for _, item := range title.Main.Trivia.Edges {
				if count >= 50 { break }
				raw := strings.ReplaceAll(item.Node.Text.PlaidHtml, "<br/>", "\n")
				raw = strings.ReplaceAll(raw, "</a>", "")
				if idx := strings.Index(raw, ">"); idx != -1 && strings.Contains(raw, "<a") { raw = raw[idx+1:] } 
				tNodes = append(tNodes, tgNode{Tag: "blockquote", Children: []any{html.UnescapeString(raw)}})
				count++
			}
		}
		if title.Main.GoofsTotal.Total > 0 {
			tNodes = append(tNodes, makeHeader(fmt.Sprintf("Goofs (%d items)", title.Main.GoofsTotal.Total)))
			count := 0
			for _, item := range title.Main.Goofs.Edges {
				if count >= 50 { break }
				tNodes = append(tNodes, tgNode{Tag: "p", Children: []any{"• " + html.UnescapeString(item.Node.Text.PlaidHtml)}})
				count++
			}
		}
		if title.Main.QuotesTotal.Total > 0 {
			tNodes = append(tNodes, makeHeader("Quotes"))
			count := 0
			for _, item := range title.Main.Quotes.Edges {
				if count >= 50 { break }
				if len(item.Node.Lines) > 0 { tNodes = append(tNodes, tgNode{Tag: "p", Children: []any{"\"" + item.Node.Lines[0].Text + "\""}}) }
				count++
			}
		}

		// Soundtrack
		if len(title.Main.Soundtrack.Edges) > 0 {
			tNodes = append(tNodes, makeHeader("Soundtrack"))
			var songs []string
			for i, s := range title.Main.Soundtrack.Edges {
				if i >= 100 { break }
				songs = append(songs, s.Node.Text)
			}
			tNodes = append(tNodes, tgNode{Tag: "p", Children: []any{strings.Join(songs, ", ")}})
		}

		// Production
		tNodes = append(tNodes, makeHeader("Production Info"))
		if len(title.Main.Production.Edges) > 0 {
			var comps []string
			for _, c := range title.Main.Production.Edges { comps = append(comps, c.Node.Company.CompanyText.Text) }
			tNodes = append(tNodes, makeRow("Companies", strings.Join(comps, ", ")))
		}
		if len(title.Main.FilmingLocations.Edges) > 0 {
			var locs []string
			for i, l := range title.Main.FilmingLocations.Edges {
				if i >= 50 { break }
				locs = append(locs, l.Node.Text)
			}
			tNodes = append(tNodes, makeRow("Filming Locations", strings.Join(locs, "; ")))
		}

		// Similar Titles
		if title.MoreLikeThisTitles != nil && len(title.MoreLikeThisTitles.Edges) > 0 {
			tNodes = append(tNodes, makeHeader("More Like This"))
			var sims []string
			for _, s := range title.MoreLikeThisTitles.Edges {
				sims = append(sims, s.Node.TitleText.Text)
			}
			tNodes = append(tNodes, tgNode{Tag: "p", Children: []any{strings.Join(sims, ", ")}})
		}

		pageURL := createTelegraphPage(title.Top.TitleText.Text+" ("+strconv.Itoa(title.Top.ReleaseYear.Year)+") - Details", tNodes)
		
		captionBuilder.WriteString(fmt.Sprintf("\n\n<a href=\"%s\">Read More...</a>", imdbURL))
		if pageURL != "" { captionBuilder.WriteString(fmt.Sprintf(" | <a href=\"%s\">Full Details</a>", pageURL)) }
	} else {
		captionBuilder.WriteString(fmt.Sprintf("\n\n<a href=\"%s\">Read More...</a>", imdbURL))
	}

	// --- 3. FOOTER ---
	trailerURL := title.Short.Trailer.EmbedURL
	if trailerURL == "" {
		trailerURL = fmt.Sprintf("https://www.youtube.com/results?search_query=%s", url.QueryEscape(title.Top.TitleText.Text+" trailer"))
	}
	captionBuilder.WriteString(fmt.Sprintf(" | <a href=\"%s\">Trailer</a>", trailerURL))

	var finalPosterURL string
	var downloadPosterURL string
	posterURL := title.Top.PrimaryImage.URL
	
	if posterURL != "" && posterURL != notAvailable {
		if strings.Contains(posterURL, "._V1_") {
			baseURL := strings.Split(posterURL, "._V1_")[0]
			finalPosterURL = baseURL + "._V1_FMjpg_UX2000_.jpg"
			downloadPosterURL = baseURL + "._V1_FMjpg_UX3000_.jpg"
		} else {
			finalPosterURL = posterURL
			downloadPosterURL = posterURL
		}
	} else {
		finalPosterURL = omdbBanner
		downloadPosterURL = ""
	}

	if downloadPosterURL != "" {
		captionBuilder.WriteString(fmt.Sprintf(" | <a href=\"%s\">Download Poster</a>", downloadPosterURL))
	}

	return finalPosterURL, captionBuilder.String(), buttons, nil
}
