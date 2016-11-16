package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/danackerson/bender-slackbot/commands"
	"github.com/jasonlvhit/gocron"
	"github.com/nlopes/slack"
)

var botID = "N/A" // U2NQSPHHD bender bot userID
var generalChannel = "C092UE0H4"
var rtm *slack.RTM

func prepareScheduler() {
	scheduler := gocron.NewScheduler()
	scheduler.Every(1).Day().At("09:39").Do(commands.ListDODroplets, rtm)
	//TODO scheduler.Every(1).Friday().At("12:39").Do(commands.ShowGames)
	<-scheduler.Start()

	// more examples: https://github.com/jasonlvhit/gocron/blob/master/example/example.go#L19
}

func main() {
	slackToken := os.Getenv("slackToken")
	api := slack.New(slackToken)
	logger := log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)
	slack.SetLogger(logger)
	api.SetDebug(false)

	go prepareScheduler() // spawn cron scheduler jobs

	rtm = api.NewRTM()
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
					(strings.Contains(ev.Msg.Text, "<@"+botID+">") || strings.HasPrefix(ev.Msg.Channel, "D")) {
					fmt.Printf("Message: %+v\n", ev.Msg)
					originalMessage := ev.Msg.Text
					parsedMessage := strings.TrimSpace(strings.Replace(originalMessage, "<@"+botID+">", "", -1)) // strip out bot's name and spaces
					commands.CheckCommand(api, rtm, ev.Msg, parsedMessage)
				}

			case *slack.PresenceChangeEvent:
				fmt.Printf("Presence Change: %+v\n", ev)

				// bug in API sets "away" sometimes when user rejoins slack :(
				/*if (ev.Presence == "away") {
				  leavingUser, _ := api.GetUserInfo(ev.User)
				  rtm.SendMessage(rtm.NewOutgoingMessage(leavingUser.Profile.FirstName+" just cheezed it!", generalChannel))
				}*/

			case *slack.LatencyReport:
				api.GetUserInfo(botID)
				//fmt.Printf("Current latency: %+v\n", ev.Value)

			case *slack.RTMError:
				fmt.Printf("Error: %s\n", ev.Error())

			case *slack.InvalidAuthEvent:
				fmt.Printf("Invalid credentials")
				break Loop

			default:
				// the gocron scheduler above communicates with the RTMbot subroutine
				// via it's builtin channel. here we check for custom events and act
				// accordingly
				if msg.Type == "ListDODroplets" {
					response := msg.Data.(string)
					params := slack.PostMessageParameters{AsUser: true}
					api.PostMessage(generalChannel, response, params)
				} else {
					// Ignore other events..
					//fmt.Printf("Unexpected %s: %+v\n", msg.Type, msg.Data)
				}
			}
		}
	}
}
