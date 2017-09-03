package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/danackerson/bender-slackbot/commands"
	"github.com/danackerson/digitalocean/common"
	"github.com/jasonlvhit/gocron"
	"github.com/nlopes/slack"
)

var botID = "N/A" // U2NQSPHHD bender bot userID

func prepareScheduler() {
	scheduler := gocron.NewScheduler()
	scheduler.Every(1).Friday().At("09:03").Do(commands.ListDODroplets, false)
	scheduler.Every(1).Friday().At("09:04").Do(commands.RaspberryPIPrivateTunnelChecks, false)
	scheduler.Every(1).Friday().At("09:05").Do(commands.CheckPiDiskSpace, "---")
	scheduler.Every(1).Day().At("05:30").Do(common.UpdateFirewall)
	//scheduler.Every(1).Friday().At("09:06").Do(commands.DownloadFile, "nfl")
	scheduler.Every(10).Minutes().Do(commands.DisconnectIdleTunnel)
	<-scheduler.Start()

	// more examples: https://github.com/jasonlvhit/gocron/blob/master/example/example.go#L19
}

func main() {
	api := slack.New(os.Getenv("slackToken"))
	logger := log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)
	slack.SetLogger(logger)
	api.SetDebug(false)

	go prepareScheduler() // spawn cron scheduler jobs

	rtm := api.NewRTM()
	commands.SetRTM(rtm)
	go rtm.ManageConnection() // spawn slack bot

Loop:
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {

			case *slack.HelloEvent:
				// Ignore hello

			case *slack.ConnectedEvent:
				botID = ev.Info.User.ID
				//rtm.SendMessage(rtm.NewOutgoingMessage("I'm back baby!", generalChannel))

			case *slack.MessageEvent:
				callerID := ev.Msg.User

				// only respond to messages sent to me by others on the same channel:
				if ev.Msg.Type == "message" && callerID != botID && ev.Msg.SubType != "message_deleted" &&
					(strings.Contains(ev.Msg.Text, "<@"+botID+">") ||
						strings.HasPrefix(ev.Msg.Channel, "D") ||
						ev.Msg.Channel == commands.SlackReportChannel) {
					logger.Printf("Message: %+v\n", ev.Msg)
					originalMessage := ev.Msg.Text
					// strip out bot's name and spaces
					parsedMessage := strings.TrimSpace(strings.Replace(originalMessage, "<@"+botID+">", "", -1))
					r, n := utf8.DecodeRuneInString(parsedMessage)
					parsedMessage = string(unicode.ToLower(r)) + parsedMessage[n:]
					commands.CheckCommand(api, ev.Msg, parsedMessage)
				}

			case *slack.PresenceChangeEvent:
				fmt.Printf("Presence Change: %+v\n", ev)

				// bug in API sets "away" sometimes when user rejoins slack :(
				/*if (ev.Presence == "away") {
				  leavingUser, _ := api.GetUserInfo(ev.User)
				  rtm.SendMessage(rtm.NewOutgoingMessage(leavingUser.Profile.FirstName+" just cheezed it!",
				  generalChannel))
				}*/

			case *slack.LatencyReport:
				api.GetUserInfo(botID)
				//logger.Printf("Current latency: %+v\n", ev.Value)

			case *slack.RTMError:
				logger.Printf("Error: %s\n", ev.Error())

			case *slack.InvalidAuthEvent:
				logger.Println("Invalid credentials")
				break Loop

			default:
				// the gocron scheduler above communicates with the RTMbot subroutine
				// via it's builtin channel. here we check for custom events and act
				// accordingly
				if msg.Type == "ListDODroplets" || msg.Type == "MoveTorrent" ||
					msg.Type == "RaspberryPIPrivateTunnelChecks" ||
					msg.Type == "CheckPiDiskSpace" {
					response := msg.Data.(string)
					params := slack.PostMessageParameters{AsUser: true}
					api.PostMessage(commands.SlackReportChannel, response, params)
				} else {
					// Ignore other events..
					//fmt.Printf("Unexpected %s: %+v\n", msg.Type, msg.Data)
				}
			}
		}
	}
}
