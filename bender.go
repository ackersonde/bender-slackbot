package main

import (
  "fmt"
  "log"
  "strings"
  "os"

  "github.com/nlopes/slack"
  "github.com/danackerson/bender-slackbot/commands"
)

var botID = "N/A" // U2NQSPHHD bender bot userID
var generalChannel = "C092UE0H4"

func main() {
  slackToken := os.Getenv("slackToken")
  api := slack.New(slackToken)
  logger := log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)
  slack.SetLogger(logger)
  api.SetDebug(false)

  rtm := api.NewRTM()
  go rtm.ManageConnection()

Loop:
  for {
    select {
    case msg := <-rtm.IncomingEvents:
      switch ev := msg.Data.(type) {
      case *slack.HelloEvent:
        // Ignore hello

      case *slack.ConnectedEvent:
        botID = ev.Info.User.ID
        rtm.SendMessage(rtm.NewOutgoingMessage("I'm back baby!", generalChannel))

      case *slack.MessageEvent:
        fmt.Printf("Message: %+v\n", ev.Msg)
        callerID := ev.Msg.User
        
        // only respond to messages sent to me by others on the same channel:
        if ev.Msg.Type == "message" && callerID != botID && ev.Msg.SubType != "message_deleted" && 
           ( strings.Contains(ev.Msg.Text, "<@"+botID+">") || strings.HasPrefix(ev.Msg.Channel, "D") ) {
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
        fmt.Printf("Current latency: %+v\n", ev.Value)

      case *slack.RTMError:
        fmt.Printf("Error: %s\n", ev.Error())

      case *slack.InvalidAuthEvent:
        fmt.Printf("Invalid credentials")
        break Loop

      default:
        // Ignore other events..
        //fmt.Printf("Unexpected: %+v\n", msg.Data)
      }
    }
  }
}
