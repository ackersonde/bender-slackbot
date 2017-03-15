package commands

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/nlopes/slack"
)

var raspberryPIIP = os.Getenv("raspberryPIIP")
var rtm *slack.RTM
var piSDCardPath = "/home/pi/torrents/"
var piUSBMountPath = "/mnt/usb_1/DLNA/torrents/"
var routerIP = "192.168.1.1"
var routerUSBMountPath = "/mnt/TOSHIBA_EXT/DLNA/torrents/"
var tranc = "tranc"

// SlackReportChannel default reporting channel for bot crons
var SlackReportChannel = os.Getenv("slackReportChannel") // C33QYV3PW is #remote_network_report

// SetRTM sets singleton
func SetRTM(rtmPassed *slack.RTM) {
	rtm = rtmPassed
}

// CheckCommand is now commented
func CheckCommand(api *slack.Client, slackMessage slack.Msg, command string) {
	args := strings.Fields(command)
	callingUserProfile, _ := api.GetUserInfo(slackMessage.User)

	if args[0] == "bb" {
		// TODO pass yesterday's date
		response := ShowBaseBallGames()
		result := "Ball games from " + response.ReadableDate + ":\n"

		// TODO split game results out into string
		for _, gameMetaData := range response.Games {
			/*
							{{ $away_team := index $gameMetaData 0}}
				      {{ $away_hp := index $gameMetaData 1}}
				      {{ $away_id := index $gameMetaData 2}}
				      {{ $away_abbrev := index $gameMetaData 3}}
				      {{ $home_team := index $gameMetaData 4}}
				      {{ $home_hp := index $gameMetaData 5}}
				      {{ $home_id := index $gameMetaData 6}}
				      {{ $home_abbrev := index $gameMetaData 7}}
				      {{ $id := index $gameMetaData 8}}
				      {{ $game_url := index $gameMetaData 10}}
							<div style="display: table-cell;padding:5px;">
				        <a href="/bbFavoriteTeam?id={{ $away_id }}" title="{{ $away_team }}"><img class="logo logo-small logo{{ $away_id }}" src="images/img_trans.gif"></a>
				      </div>
				      <div style="display:table-cell;padding:5px;text-align:center;vertical-align:middle;"><a href="/bbStream?url={{ $game_url }}">{{ $away_abbrev }}@{{ $home_abbrev }}</a></div>
				      <div style="display:table-cell;padding:5px;">
				        <a href="/bbFavoriteTeam?id={{ $home_id }}" title="{{ $home_team }}"><img class="logo logo-small logo{{ $home_id }}" src="images/img_trans.gif"></a>
				      </div>
			*/
			// TODO: paint 2 urls, one for watching, one for telling ackerson.de
			// server to download appropriately named bbGame.mp4 to ~/bb_games/
			// TODO2: send ~/bb_games/bbGame.mp4 to Join Push app to send file to cell
			watchURL := "<" + gameMetaData[10] + "|" + gameMetaData[0] + " @ " + gameMetaData[4] + ">    "
			downloadURL := "<https://ackerson.de/bb_download?gameTitle=" + gameMetaData[2] + "-" + gameMetaData[6] + "__" + response.ReadableDate + "&gameURL=" + gameMetaData[10] + " | [ send to " + callingUserProfile.Name + "'s :smartphone: ]>"

			result += watchURL + downloadURL + "\n"
		}

		params := slack.PostMessageParameters{AsUser: true}
		api.PostMessage(slackMessage.Channel, result, params)
	} else if args[0] == "bbg" {
		params := slack.PostMessageParameters{AsUser: true}

		if len(args) > 1 {
			result := GetBaseBallGame(args[1])
			api.PostMessage(slackMessage.Channel, result, params)
		} else {
			api.PostMessage(slackMessage.Channel, "Please provide Game ID from `bb` cmd!", params)
		}
	} else if args[0] == "ms" {
		response := "Failed to restart miniDLNA on :asus:"

		result := ResetMediaServer()
		if result {
			response = "Successfully restarted miniDLNA on :asus:"
		}

		params := slack.PostMessageParameters{AsUser: true}
		api.PostMessage(slackMessage.Channel, response, params)
	} else if args[0] == "do" {
		response := ListDODroplets(true)
		params := slack.PostMessageParameters{AsUser: true}
		api.PostMessage(slackMessage.Channel, response, params)
	} else if args[0] == "dd" {
		params := slack.PostMessageParameters{AsUser: true}

		if len(args) > 1 {
			number, err := strconv.Atoi(args[1])
			if err != nil {
				api.PostMessage(slackMessage.Channel, "Invalid integer value for ID!", params)
			} else {
				result := DeleteDODroplet(number)
				api.PostMessage(slackMessage.Channel, result, params)
			}
		} else {
			api.PostMessage(slackMessage.Channel, "Please provide Droplet ID from `do` cmd!", params)
		}
	} else if args[0] == "fsck" {
		if runningFritzboxTunnel() {
			response := ""

			if len(args) > 1 {
				path := strings.Join(args[1:], " ")
				response += CheckPiDiskSpace(path)
			} else {
				response += CheckPiDiskSpace("")
			}

			rtm.SendMessage(rtm.NewOutgoingMessage(response, slackMessage.Channel))
		}
	} else if args[0] == "mv" || args[0] == "rm" {
		response := ""
		if len(args) > 1 {
			if runningFritzboxTunnel() {
				path := strings.Join(args[1:], " ")
				if args[0] == "rm" {
					response = DeleteTorrentFile(path)
				} else {
					MoveTorrentFile(path)
				}

				rtm.SendMessage(rtm.NewOutgoingMessage(response, slackMessage.Channel))
			}
		} else {
			rtm.SendMessage(rtm.NewOutgoingMessage("Please provide a filename", slackMessage.Channel))
		}
	} else if args[0] == "torq" {
		var response string
		cat := 0
		if len(args) > 1 {
			if args[1] == "nfl" {
				cat = 200
			} else if args[1] == "ubuntu" {
				cat = 300
			}

			searchString := strings.Join(args, " ")
			searchString = strings.TrimLeft(searchString, "torq")
			fmt.Println("searching for: " + searchString)
			_, response = SearchFor(searchString, Category(cat))
		} else {
			_, response = SearchFor("", Category(cat))
		}
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
	} else if args[0] == "trans" || args[0] == "trand" || args[0] == tranc {
		if runningFritzboxTunnel() {
			response := torrentCommand(args)
			rtm.SendMessage(rtm.NewOutgoingMessage(response, slackMessage.Channel))
		}
	} else if args[0] == "help" {
		response := ":sun_behind_rain_cloud: `sw`: Schwabhausen weather\n" +
			":do_droplet: `do|dd <id>`: show|delete DigitalOcean droplet(s)\n" +
			":closed_lock_with_key: `vpn[c|s|d]`: [C]onnect, [S]tatus, [D]rop VPN tunnel to Fritz!Box\n" +
			":pirate_bay: `torq <search term>`\n" +
			":openvpn: `ovpn`: show status of PrivateTunnel on :raspberry_pi:\n" +
			":transmission: `tran[c|s|d]`: [C]reate <URL>, [S]tatus, [D]elete <ID> torrents on :raspberry_pi:\n" +
			":floppy_disk: `fsck`: show disk space on :raspberry_pi:\n" +
			":recycle: `rm(|mv) <filename>` from :raspberry_pi: (to `" + routerUSBMountPath + "` on :asus:)\n" +
			":movie_camera: `ms`: restart miniDLNA media server on :asus:\n" +
			":baseball: `bb`: show yesterday's baseball games\n"
		params := slack.PostMessageParameters{AsUser: true}
		api.PostMessage(slackMessage.Channel, response, params)
	} else {
		rtm.SendMessage(rtm.NewOutgoingMessage("whaddya say <@"+callingUserProfile.Name+">? Try `help` instead...",
			slackMessage.Channel))
	}
}

/* DownloadFile is now exported
func DownloadFile(search string) {
	torrents, results := SearchFor(search, 200)
	for num, torrent := range torrents {
		if num < 20 {
			fmt.Println(torrent.Title)
			// TODO figure out date of game and compare to today's date
			// type1: NFL.2016.RS.W12.(28 nov).GB
			// type2: NFL.2016.12.11.Cowboys
			// type3: NFL.2016.RS.W13.KC.

		}
	}

	var tor []string
	tor[0] = tranc
	tor[1] = results
	if runningFritzboxTunnel() {
		trans := torrentCommand(tor)
		fmt.Println(trans)
	}
}*/
