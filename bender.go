package main

import (
	"log"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/danackerson/bender-slackbot/commands"
	"github.com/jasonlvhit/gocron"
	"github.com/nlopes/slack"
)

var botID = "N/A" // U2NQSPHHD bender bot userID

func prepareScheduler() {
	//gocron.Every(1).Friday().At("09:03").Do(commands.ListDODroplets, false)
	gocron.Every(1).Friday().At("09:04").Do(commands.RaspberryPIPrivateTunnelChecks, false)
	gocron.Every(1).Friday().At("09:05").Do(commands.CheckTorrentsDiskSpace, "---")
	gocron.Every(1).Friday().At("09:05").Do(commands.CheckPlexDiskSpace, "---")
	//gocron.Every(1).Day().At("05:30").Do(common.UpdateFirewall)
	//gocron.Every(1).Day().At("17:30").Do(commands.ShowBBGames, false, "")
	<-gocron.Start()

	// more examples: https://github.com/jasonlvhit/gocron/blob/master/example/example.go#L19
}

func main() {
	api := slack.New(os.Getenv("CTX_SLACK_API_TOKEN"))
	logger := log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)

	slack.OptionLog(logger)
	slack.OptionDebug(false)

	go prepareScheduler() // spawn cron scheduler jobs

	rtm := api.NewRTM()
	commands.SetRTM(rtm)
	go rtm.ManageConnection() // spawn slack bot

	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {

			case *slack.ConnectedEvent:
				botID = ev.Info.User.ID

			case *slack.MessageEvent:
				originalMessage := ev.Msg.Text
				callerID := ev.Msg.User
				userInfo, _ := rtm.GetUserInfo(callerID)

				if userInfo == nil {
					logger.Printf("rcvd %s from %v\n", originalMessage, ev.Msg.User)
				} else {
					logger.Printf("rcvd %s from: %s(%s)\n", originalMessage, userInfo.Name, userInfo.ID)
				}

				// only respond to messages sent to me by others on the same channel:
				if ev.Msg.Type == "message" && callerID != botID && ev.Msg.SubType != "message_deleted" &&
					(strings.Contains(ev.Msg.Text, "<@"+botID+">") ||
						strings.HasPrefix(ev.Msg.Channel, "D") ||
						ev.Msg.Channel == commands.SlackReportChannel) {
					// strip out bot's name and spaces
					parsedMessage := strings.TrimSpace(strings.Replace(originalMessage, "<@"+botID+">", "", -1))
					r, n := utf8.DecodeRuneInString(parsedMessage)
					parsedMessage = string(unicode.ToLower(r)) + parsedMessage[n:]

					var userName string
					if userInfo == nil {
						userName = "algo-build-bot"
					} else {
						userName = userInfo.Name
					}
					logger.Printf("%s: %v\n", userName, parsedMessage)

					commands.CheckCommand(api, ev.Msg, parsedMessage)
				}

			case *slack.RTMError:
				logger.Printf("Error: %s\n", ev.Error())

			case *slack.InvalidAuthEvent:
				logger.Println("Invalid credentials")
				break

			default:
				// the gocron scheduler above communicates with the RTMbot subroutine
				// via it's builtin channel. here we check for custom events and act
				// accordingly
				if msg.Type == "ListDODroplets" || msg.Type == "MoveTorrent" ||
					msg.Type == "RaspberryPIPrivateTunnelChecks" ||
					msg.Type == "CheckPiDiskSpace" || msg.Type == "ShowBBGames" {
					response := msg.Data.(string)
					params := slack.MsgOptionAsUser(true)

					if msg.Type == "MoveTorrent" {
						api.PostMessage(commands.SlackReportChannel, slack.MsgOptionText("DONE moving files. Enjoy your :movie_camera: & :popcorn:!", true), params)
					} else {
						api.PostMessage(commands.SlackReportChannel, slack.MsgOptionText(response, false), params)
					}
				} else {
					// Ignore other events..
					// fmt.Printf("Unexpected %s: %+v\n", msg.Type, msg.Data)
				}
			}
		}
	}
}
