package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/nlopes/slack"
)

func execCmd(command string) {
	cmd := exec.Command(command)
	err := cmd.Start()
	if err != nil {
		os.Stderr.WriteString(err.Error())
	}
	cmd.Wait()
}

// CheckCommand is now commented
func CheckCommand(api *slack.Client, rtm *slack.RTM, slackMessage slack.Msg, command string) {
	fmt.Printf("rcvd cmd: %s", command)

	if command == "sw" {
		response := ":partly_sunny_rain: <https://www.wunderground.com/cgi-bin/findweather/getForecast?query=48.3,11.35#forecast-graph|10-day forecast Schwabhausen>"
		params := slack.PostMessageParameters{AsUser: true}
		//rtm.SendMessage(rtm.NewOutgoingMessage(response, slackMessage.Channel))
		api.PostMessage(slackMessage.Channel, response, params)
	} else if command == "vpnc" {
		go execCmd("/usr/sbin/vpnc-connect fritzbox")
		rtm.SendMessage(rtm.NewOutgoingMessage("tunnel up", slackMessage.Channel))
	} else if command == "disc" {
		go execCmd("/usr/sbin/vpnc-disconnect")
		rtm.SendMessage(rtm.NewOutgoingMessage("tunnel down", slackMessage.Channel))
	} else {
		callingUserProfile, _ := api.GetUserInfo(slackMessage.User)
		rtm.SendMessage(rtm.NewOutgoingMessage("whaddya say <@"+callingUserProfile.Name+">? "+command+"?", slackMessage.Channel))
	}
}
