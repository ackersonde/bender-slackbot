package main

import (
  "fmt"
  "log"
  "strings"
  "os"

  "github.com/nlopes/slack"
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
      //fmt.Print("Event Received: ")
      switch ev := msg.Data.(type) {
      case *slack.HelloEvent:
        // Ignore hello

      case *slack.ConnectedEvent:
        //fmt.Println("Infos:", ev.Info)
        //fmt.Println("Connection counter:", ev.ConnectionCount)
        botID = ev.Info.User.ID
        // TODO - reenable on version 0.1
        rtm.SendMessage(rtm.NewOutgoingMessage("I'm back baby!", generalChannel))

      case *slack.MessageEvent:
        fmt.Printf("Message: %+v\n", ev.Msg)
        // only react to messages to me and on the same channel!
        //botUser, _ := api.GetUserInfo(botID)
        callingUser, _ := api.GetUserInfo(ev.Msg.User)
        fmt.Printf("botUser: %+v\n", botID)
        fmt.Printf("calUser: %+v\n", ev.Msg.User)
        if ev.Msg.Type == "message" && ev.Msg.User != botID && ev.Msg.SubType != "message_deleted" && 
           ( strings.Contains(ev.Msg.Text, "<@"+botID+">") || strings.HasPrefix(ev.Msg.Channel, "D") ) {
          originalMessage := ev.Msg.Text
          parsedMessage := strings.Replace(originalMessage, "<@"+botID+">", "", -1) // strip out bot's name from cmd
          rtm.SendMessage(rtm.NewOutgoingMessage("whaddya say <@"+callingUser.Name+">? "+parsedMessage+"?", ev.Msg.Channel))
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
