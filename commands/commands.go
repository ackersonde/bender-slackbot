package commands

import (
  "fmt"
 
  "github.com/nlopes/slack"
)

// TestMessage is now commented
func CheckCommand(api *slack.Client, rtm *slack.RTM, slackMessage slack.Msg, command string) {
  fmt.Printf("rcvd cmd: %s", command)
  
  if (command == "sw") {
    response := ":partly_sunny_rain: <https://www.wunderground.com/cgi-bin/findweather/getForecast?query=48.3,11.35#forecast-graph|10-day forecast Schwabhausen>"
    params := slack.PostMessageParameters{ AsUser: true }
    rtm.SendMessage(rtm.NewOutgoingMessage(response, slackMessage.Channel))
    api.PostMessage(slackMessage.Channel, response, params)
  } else {
    callingUserProfile, _ := api.GetUserInfo(slackMessage.User)
    rtm.SendMessage(rtm.NewOutgoingMessage("whaddya say <@"+callingUserProfile.Name+">? "+command+"?", slackMessage.Channel))
  }
}