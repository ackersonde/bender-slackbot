package commands

import (
	"net/url"
	"strings"
	"time"

	"github.com/ackersonde/ackerson.de-go/baseball"
	"github.com/slack-go/slack"
)

// FavGames is now commented
type FavGames struct {
	FavGamesList []baseball.GameDay
	FavTeam      baseball.Team
}

// GameDay is now commented
type GameDay struct {
	Date         string
	ReadableDate string
	Games        map[int][]string
}

var homePageMap map[int]baseball.Team

// GetBaseBallGame is now commented
func GetBaseBallGame(gameID string) string {
	return baseball.FetchGameURLFromID(gameID)
}

// ShowBBGames is now commented
func ShowBBGames(fromDate string) string {
	if fromDate == "" {
		yesterday := time.Now().AddDate(0, 0, -1)
		fromDate = yesterday.Format("2006/month_01/day_02")
	}
	response := ShowBaseBallGames(fromDate)
	result := "Ball games from " + response.ReadableDate + ":\n"

	for _, gameMetaData := range response.Games {
		watchURL := "<" + gameMetaData[10] + "|" + gameMetaData[0] + " @ " + gameMetaData[4] + ">"
		downloadURL := "<https://ackerson.de/bb_download?gameTitle=" + gameMetaData[2] +
			"-" + gameMetaData[6] + "__" + strings.ReplaceAll(response.ReadableDate, " ", "%20") + "&gameURL=" +
			url.QueryEscape(gameMetaData[10]) + "|(download) :red_dot:>  "

		Logger.Printf("WTF: %s", downloadURL)
		result += downloadURL + watchURL + "\n"
	}

	return result
}

// ShowBBGamesCron called from scheduler
func ShowBBGamesCron(fromDate string) {
	api.PostMessage(SlackReportChannel, slack.MsgOptionText(
		ShowBBGames(fromDate), false), slack.MsgOptionAsUser(true))
}

// ShowBaseBallGames now commented
func ShowBaseBallGames(fromDate string) baseball.GameDay {
	homePageMap = baseball.InitHomePageMap()

	offset := ""

	gameDayListing := baseball.GameDayListingHandler(fromDate, offset, homePageMap)

	return gameDayListing
}
