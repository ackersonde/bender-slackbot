package commands

import (
	"github.com/danackerson/ackerson.de-go/baseball"
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
	// TODO download this damn thing to /home/ackersond/bb_games/
	return baseball.FetchGameURLFromID(gameID)
}

// ShowBaseBallGames now commented
func ShowBaseBallGames() baseball.GameDay {
	homePageMap = baseball.InitHomePageMap()

	date1 := ""
	offset := ""

	if date1 == "" {
		//TODO 2016 is over - make default Game 7, 2016 World Series
		date1 = "year_2016/month_11/day_02"
		offset = "0"
	}

	gameDayListing := baseball.GameDayListingHandler(date1, offset, homePageMap)

	return gameDayListing
}
