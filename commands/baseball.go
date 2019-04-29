package commands

import (
	"time"

	"github.com/danackerson/ackerson.de-go/baseball"
	"github.com/nlopes/slack"
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
func ShowBBGames(userCall bool, fromDate string) string {
	if fromDate == "" {
		yesterday := time.Now().AddDate(0, 0, -1)
		fromDate = yesterday.Format("2006/month_01/day_02")
	}
	response := ShowBaseBallGames(fromDate)
	result := "Ball games from " + response.ReadableDate + ":\n"

	for _, gameMetaData := range response.Games {
		watchURL := "<" + gameMetaData[10] + "|" + gameMetaData[0] + " @ " + gameMetaData[4] + ">    "
		downloadURL := "<https://ackerson.de/bb_download?fileType=bb&gameTitle=" + gameMetaData[2] + "-" + gameMetaData[6] + "__" + response.ReadableDate + "&gameURL=" + gameMetaData[10] + " | :smartphone:>"

		result += watchURL + downloadURL + "\n"
	}

	if !userCall {
		rtm.IncomingEvents <- slack.RTMEvent{Type: "ShowBBGames", Data: result}
	}

	return result
}

// ShowBaseBallGames now commented
func ShowBaseBallGames(fromDate string) baseball.GameDay {
	homePageMap = baseball.InitHomePageMap()

	offset := ""

	gameDayListing := baseball.GameDayListingHandler(fromDate, offset, homePageMap)

	return gameDayListing
}
