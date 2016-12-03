package commands

import (
	"os"
	"strings"

	"github.com/nlopes/slack"
)

var raspberryPIIP = os.Getenv("raspberryPIIP")
var rtm *slack.RTM

// SlackReportChannel default reporting channel for bot crons
var SlackReportChannel = os.Getenv("slackReportChannel") // C33QYV3PW is #remote_network_report

// SetRTM sets singleton
func SetRTM(rtmPassed *slack.RTM) {
	rtm = rtmPassed
}

// CheckCommand is now commented
func CheckCommand(api *slack.Client, slackMessage slack.Msg, command string) {
	args := strings.Fields(command)
	if args[0] == "do" {
		response := ListDODroplets(true)
		params := slack.PostMessageParameters{AsUser: true}
		api.PostMessage(slackMessage.Channel, response, params)
	} else if args[0] == "torq" {
		cat := 207
		if args[1] == "nfl" {
			cat = 200
		} else if args[1] == "ubuntu" {
			cat = 300
		}
		response := SearchFor(args[1], Category(cat))
		params := slack.PostMessageParameters{AsUser: true}
		api.PostMessage(slackMessage.Channel, response, params)
	} else if args[0] == "ovpn" {
		response := RaspberryPIPrivateTunnelChecks(true)
		rtm.SendMessage(rtm.NewOutgoingMessage(response, slackMessage.Channel))
	} else if args[0] == "sw" {
		response := ":partly_sunny_rain: <https://www.wunderground.com/cgi-bin/findweather/getForecast?query=" +
			"48.3,11.35#forecast-graph|10-day forecast Schwabhausen>"
		params := slack.PostMessageParameters{AsUser: true}
		api.PostMessage(slackMessage.Channel, response, params)
	} else if args[0] == "vpnc" {
		result := vpnTunnelCmds("/usr/sbin/vpnc-connect", "fritzbox")
		rtm.SendMessage(rtm.NewOutgoingMessage(result, slackMessage.Channel))
	} else if args[0] == "vpnd" {
		result := vpnTunnelCmds("/usr/sbin/vpnc-disconnect")
		rtm.SendMessage(rtm.NewOutgoingMessage(result, slackMessage.Channel))
	} else if args[0] == "vpns" {
		result := vpnTunnelCmds("status")
		rtm.SendMessage(rtm.NewOutgoingMessage(result, slackMessage.Channel))
	} else if args[0] == "trans" || args[0] == "trand" || args[0] == "tranc" {
		if runningFritzboxTunnel() {
			response := torrentCommand(args)
			rtm.SendMessage(rtm.NewOutgoingMessage(response, slackMessage.Channel))
		}
	} else if args[0] == "help" {
		response := ":sun_behind_rain_cloud: `sw`: Schwabhausen weather\n" +
			":do_droplet: `do`: show current DigitalOcean droplets\n" +
			":closed_lock_with_key: `vpn[c|s|d]`: [C]onnect, [S]tatus, [D]rop VPN tunnel to Fritz!Box\n" +
			":openvpn: `ovpn`: show status of PrivateTunnel on :raspberry_pi:\n" +
			":transmission: `tran[c|s|d]`: [C]reate <URL>, [S]tatus, [D]elete <ID> torrents on :raspberry_pi:\n" +
			":pirate_bay: `torq <search term>`\n"
		params := slack.PostMessageParameters{AsUser: true}
		api.PostMessage(slackMessage.Channel, response, params)
	} else {
		callingUserProfile, _ := api.GetUserInfo(slackMessage.User)
		rtm.SendMessage(rtm.NewOutgoingMessage("whaddya say <@"+callingUserProfile.Name+">? Try `help` instead...",
			slackMessage.Channel))
	}
}
