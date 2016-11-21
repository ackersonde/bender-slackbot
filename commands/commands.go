package commands

import (
	"strings"

	"github.com/nlopes/slack"
)

var raspberryPIIP = "192.168.178.38"
var rtm *slack.RTM

// SetRTM sets singleton
func SetRTM(rtmPassed *slack.RTM) {
	rtm = rtmPassed
}

// CheckCommand is now commented
func CheckCommand(api *slack.Client, slackMessage slack.Msg, command string) {
	args := strings.Fields(command)
	if args[0] == "do" {
		ListDODroplets()
	} else if args[0] == "ovpn" {
		response := ":closed_lock_with_key: No tunnel exists! Try `vpnc` first..."
		tunnelStatus := vpnTunnelCmds("status")
		if strings.Contains(tunnelStatus, "inet 192.168.178.201/32 scope global tun0") {
			RaspberryPIPrivateTunnelChecks()
		} else {
			rtm.SendMessage(rtm.NewOutgoingMessage(response, slackMessage.Channel))
		}
	} else if args[0] == "sw" {
		response := ":partly_sunny_rain: <https://www.wunderground.com/cgi-bin/findweather/getForecast?query=48.3,11.35#forecast-graph|10-day forecast Schwabhausen>"
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
	} else if args[0] == "trans" {
		result := torrentCommand(args)
		rtm.SendMessage(rtm.NewOutgoingMessage(result, slackMessage.Channel))
	} else if args[0] == "trand" {
		result := torrentCommand(args)
		rtm.SendMessage(rtm.NewOutgoingMessage(result, slackMessage.Channel))
	} else if args[0] == "tranc" {
		result := torrentCommand(args)
		rtm.SendMessage(rtm.NewOutgoingMessage(result, slackMessage.Channel))
	} else if args[0] == "help" {
		response := ":sun_behind_rain_cloud: `sw`: Schwabhausen weather\n" +
			":do_droplet: `do`: show current DigitalOcean droplets\n" +
			":closed_lock_with_key: `vpn[c|s|d]`: [C]onnect, [S]tatus, [D]rop VPN tunnel to fritz.box\n" +
			":transmission: `tran[c|s|d]`: [C]reate <URL>, [S]tatus, [D]elete <ID> torrents on RaspberryPI\n"
		params := slack.PostMessageParameters{AsUser: true}
		api.PostMessage(slackMessage.Channel, response, params)
	} else {
		callingUserProfile, _ := api.GetUserInfo(slackMessage.User)
		rtm.SendMessage(rtm.NewOutgoingMessage("whaddya say <@"+callingUserProfile.Name+">? Try `help` instead...", slackMessage.Channel))
	}
}
