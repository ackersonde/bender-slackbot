package commands

import (
  "fmt"
 
  "github.com/nlopes/slack"
)

// TestMessage is now commented
func CheckCommand(api *slack.Client, rtm *slack.RTM, slackMessage slack.Msg, command string) {
  callingUserProfile, _ := api.GetUserInfo(slackMessage.User)
  fmt.Printf("rcvd cmd: %s", command)
  
  if (command == "sw") {
    response := ":partly_sunny_rain: <https://www.wunderground.com/cgi-bin/findweather/getForecast?query=48.3,11.35#forecast-graph|10-day forecast Schwabhausen>"
    params := slack.PostMessageParameters{ AsUser: true }
    api.PostMessage(slackMessage.Channel, response, params)
  } else {
    rtm.SendMessage(rtm.NewOutgoingMessage("whaddya say <@"+callingUserProfile.Name+">? "+command+"?", slackMessage.Channel))
  }

  /* example of sending a POST with URLs
  resp = "See the top 3 'users' on our web properties. It comes from nginx logs sent thru CloudWatch to our <%s|ElasticSearch> cluster.\nDefault search is last 30mins, but you can specify an integer param for a different range in minutes." % aws_elk_dash
  params = {"channel": msg_context.channel, "text": resp, "as_user": 'true'}
  slack_client.api_call('chat.postMessage', **params) */
}