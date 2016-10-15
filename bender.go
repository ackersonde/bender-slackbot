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

  for {
    select {
    case msg := <-rtm.IncomingEvents:
      switch ev := msg.Data.(type) {
        case *slack.ConnectedEvent:
          botID = ev.Info.User.ID
          rtm.SendMessage(rtm.NewOutgoingMessage("I'm back baby!", generalChannel))

        case *slack.MessageEvent:
          callerID := ev.Msg.User
          
          // only respond to messages sent to me by others on the same channel:
          if ev.Msg.Type == "message" && callerID != botID && ev.Msg.SubType != "message_deleted" && 
             ( strings.Contains(ev.Msg.Text, "<@"+botID+">") || strings.HasPrefix(ev.Msg.Channel, "D") ) {
            fmt.Printf("Message: %+v\n", ev.Msg)
            originalMessage := ev.Msg.Text
            parsedMessage := strings.TrimSpace(strings.Replace(originalMessage, "<@"+botID+">", "", -1)) // strip out bot's name and spaces
            commands.CheckCommand(api, rtm, ev.Msg, parsedMessage)
          }

        case *slack.PresenceChangeEvent:
          fmt.Printf("Presence Change: %+v\n", ev)

        case *slack.RTMError:
          fmt.Printf("Error: %s\n", ev.Error())
      }
    }
  }
}
