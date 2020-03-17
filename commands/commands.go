package commands

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/danackerson/digitalocean/common"
	"github.com/nlopes/slack"
)

var rtm *slack.RTM
var joinAPIKey = os.Getenv("CTX_JOIN_API_KEY")
var vpnGateway = os.Getenv("CTX_VPNC_GATEWAY")
var circleCIBuildNum = os.Getenv("CIRCLE_BUILD_NUM")

var circleCIDoAlgoURL = "https://circleci.com/api/v1.1/project/github/danackerson/do-algo"
var circleCITokenParam = "?circle-token=" + os.Getenv("CTX_CIRCLECI_API_TOKEN")

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
	params := slack.MsgOptionAsUser(true)

	if args[0] == "yt" {
		if len(args) > 1 {
			// strip '<>' off url
			downloadURL := strings.Trim(args[1], "<>")
			uri, err := url.ParseRequestURI(downloadURL)
			log.Printf("parsed %s from %s", uri.RequestURI(), downloadURL)
			if err != nil {
				api.PostMessage(slackMessage.Channel,
					slack.MsgOptionText(
						"Invalid URL for downloading! ("+err.Error()+
							")", true), params)
			} else {
				if downloadYoutubeVideo(uri.String()) {
					api.PostMessage(slackMessage.Channel,
						slack.MsgOptionText(
							"Requested YouTube video...", true), params)
				} else {
					api.PostMessage(slackMessage.Channel,
						slack.MsgOptionText(
							"Unable to download YouTube video...", true), params)
				}
			}
		} else {
			api.PostMessage(slackMessage.Channel,
				slack.MsgOptionText("Please provide YouTube video URL!", true), params)
		}
	} else if args[0] == "bb" {
		result := ""
		dateString := ""

		if len(args) > 1 {
			// TODO: use https://github.com/olebedev/when for Natural Language processing
			gameDate, err := time.Parse("2006-01-02", args[1])
			dateString = gameDate.Format("2006/month_01/day_02")

			if err != nil {
				result = "Couldn't figure out date '" + args[1] + "'. Try `help`"
				api.PostMessage(slackMessage.Channel, slack.MsgOptionText(result, false), params)
				return
			}
		}
		result = ShowBBGames(true, dateString)
		api.PostMessage(slackMessage.Channel, slack.MsgOptionText(result, false), params)
	} else if args[0] == "do" {
		response := ListDODroplets(true)
		api.PostMessage(slackMessage.Channel, slack.MsgOptionText(response, false), params)
	} else if args[0] == "dd" {
		if len(args) > 1 {
			number, err := strconv.Atoi(args[1])
			if err != nil {
				api.PostMessage(slackMessage.Channel, slack.MsgOptionText("Invalid integer value for ID!", true), params)
			} else {
				result := common.DeleteDODroplet(number)
				api.PostMessage(slackMessage.Channel, slack.MsgOptionText(result, true), params)
			}
		} else {
			api.PostMessage(slackMessage.Channel, slack.MsgOptionText("Please provide Droplet ID from `do` cmd!", true), params)
		}
	} else if args[0] == "fsck" {
		response := ""
		if len(args) > 1 {
			path := strings.Join(args[1:], " ")
			response += CheckMediaDiskSpace(path)
			response += CheckServerDiskSpace(path)
		} else {
			response += CheckMediaDiskSpace("")
			response += CheckServerDiskSpace("")
		}
		rtm.SendMessage(rtm.NewOutgoingMessage(response, slackMessage.Channel))
	} else if args[0] == "wgs" {
		response := wireguardShow()
		rtm.SendMessage(rtm.NewOutgoingMessage(response, slackMessage.Channel))
	} else if args[0] == "mv" {
		response := ""
		if len(args) == 3 &&
			(strings.HasPrefix(args[2], "movies") ||
				strings.HasPrefix(args[2], "tv")) {
			sourceFile := scrubParamOfHTTPMagicCrap(args[1])
			destinationDir := args[2]
			if strings.Contains(destinationDir, "..") || strings.HasPrefix(destinationDir, "/") {
				msg := fmt.Sprintln("Please prefix destination w/ either `[movies|tv]`")
				rtm.IncomingEvents <- slack.RTMEvent{Type: "MoveTorrent", Data: msg}
			} else if strings.Contains(sourceFile, "..") || strings.HasPrefix(sourceFile, "/") {
				msg := fmt.Sprintf("Please specify file to move relative to `%s/torrents/`\n", piPlexPath)
				rtm.IncomingEvents <- slack.RTMEvent{Type: "MoveTorrent", Data: msg}
			} else {
				MoveTorrentFile(api, sourceFile, destinationDir)
				rtm.SendMessage(rtm.NewOutgoingMessage(response, slackMessage.Channel))
			}
		} else {
			rtm.SendMessage(rtm.NewOutgoingMessage("Please provide a src file and destination [e.g. `movies` or `tv`]", slackMessage.Channel))
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
			searchString = strings.TrimPrefix(searchString, "torq ")
			_, response = SearchFor(searchString, Category(cat))
		} else {
			_, response = SearchFor("", Category(cat))
		}
		api.PostMessage(slackMessage.Channel, slack.MsgOptionText(response, false), params)
	} else if args[0] == "vpns" {
		vpnCountry := "DE"
		if len(args) > 1 {
			vpnCountry = strings.ToUpper(args[1])
		}
		response := VpnPiTunnelChecks(vpnCountry, true)
		rtm.SendMessage(rtm.NewOutgoingMessage(response, slackMessage.Channel))
	} else if args[0] == "vpnc" {
		if len(args) > 1 {
			vpnServerDomain := strings.ToLower(scrubParamOfHTTPMagicCrap(args[1]))
			response := updateVpnPiTunnel(vpnServerDomain)
			rtm.SendMessage(rtm.NewOutgoingMessage(response, slackMessage.Channel))
		} else {
			rtm.SendMessage(rtm.NewOutgoingMessage("Please provide a new VPN server (hint: output from `vpns`)", slackMessage.Channel))
		}
	} else if args[0] == "version" {
		response := ":circleci: <https://circleci.com/gh/danackerson/bender-slackbot/" +
			circleCIBuildNum + "|" + circleCIBuildNum + ">"
		api.PostMessage(slackMessage.Channel, slack.MsgOptionText(response, false), params)
	} else if args[0] == "sw" {
		response := ":partly_sunny_rain: <https://darksky.net/forecast/48.3028,11.3591/ca24/en#week|7-day forecast Schwabhausen>"
		api.PostMessage(slackMessage.Channel, slack.MsgOptionText(response, false), params)
	} else if args[0] == "trans" || args[0] == "trand" || args[0] == "tranc" || args[0] == "tranp" {
		response := torrentCommand(args)
		api.PostMessage(slackMessage.Channel, slack.MsgOptionText(response, false), params)
	} else if args[0] == "mvv" {
		response := "<https://img.srv2.de/customer/sbahnMuenchen/newsticker/newsticker.html|Aktuelles>"
		response += " | <" + mvvRoute("Schwabhausen", "München, Hauptbahnhof") + "|Going in>"
		response += " | <" + mvvRoute("München, Hauptbahnhof", "Schwabhausen") + "|Going home>"

		api.PostMessage(slackMessage.Channel, slack.MsgOptionText(response, false), params)
	} else if args[0] == "www" {
		fritzBox := ":fritzbox: <https://fritz.ackerson.de/|fritz.box> | "
		fritzBox += ":traefik: <https://monitor.ackerson.de/dashboard/#/ | traefik> | "
		pi4 := ":k8s: <https://dash.ackerson.de/#/overview?namespace=default|k8s> | "
		pi4 += ":netdata: <http://pi4:19999/#menu_cpu_submenu_utilization;theme=slate;help=true|netdata>\n"
		pi4 += ":pihole: <https://hole.ackerson.de/admin/|pi.hole> | "
		vpnpi := ":transmission: <http://vpnpi:9091/transmission/web/|trans> | "
		vpnpi += ":plex: <http://vpnpi:32400/web/index.html#|plex> | "
		vpnpi += ":traefik: <https://api-wc-gcp.ackerson.de/dashboard/#/ | weechat>\n"

		response := fritzBox + pi4 + vpnpi
		api.PostMessage(slackMessage.Channel, slack.MsgOptionText(response, false), params)
	} else if args[0] == "help" {
		response :=
			":sun_behind_rain_cloud: `sw`: Schwabhausen weather\n" +
				":metro: `mvv`: Status | Trip In | Trip Home\n" +
				":baseball: `bb <YYYY-MM-DD>`: show baseball games from given date (default yesterday)\n" +
				//":do_droplet: `do|dd <id>`: show|delete DigitalOcean droplet(s)\n" +
				":wireguard: `wgs`: show current wireguard peer status\n" +
				":protonvpn: `vpn[s|c]`: [S]how status of VPN on :raspberry_pi:, [C]hange VPN to best in given country or DE\n" +
				":pirate_bay: `torq <search term>`\n" +
				":transmission: `tran[c|p|s|d]`: [C]reate <URL>, [P]aused <URL>, [S]tatus, [D]elete <ID> torrents on :raspberry_pi:\n" +
				":movie_camera: `mv " + piPlexPath + "/torrents/<filename> [movies|tv/(<path>)]`\n" +
				":floppy_disk: `fsck`: show disk space on :raspberry_pi:\n" +
				":youtube: `yt <video url>`: Download Youtube video to Papa's handy\n" +
				":circleci: `version`: Which build number is this Bender?\n"
		api.PostMessage(slackMessage.Channel, slack.MsgOptionText(response, true), params)
	} else if callingUserProfile != nil {
		rtm.SendMessage(rtm.NewOutgoingMessage("whaddya say <@"+callingUserProfile.Name+">? Try `help` instead...",
			slackMessage.Channel))
	} else {
		log.Printf("No Command found: %s", slackMessage.Text)
	}
}

func mvvRoute(origin string, destination string) string {
	loc, _ := time.LoadLocation("Europe/Berlin")
	date := time.Now().In(loc)

	yearObj := date.Year()
	monthObj := int(date.Month())
	dayObj := date.Day()
	hourObj := date.Hour()
	minuteObj := date.Minute()

	month := strconv.Itoa(monthObj)
	hour := strconv.Itoa(hourObj)
	day := strconv.Itoa(dayObj)
	minute := strconv.Itoa(minuteObj)
	year := strconv.Itoa(yearObj)

	return "http://efa.mvv-muenchen.de/mvv/XSLT_TRIP_REQUEST2?&language=de" +
		"&anyObjFilter_origin=0&sessionID=0&itdTripDateTimeDepArr=dep&type_destination=any" +
		"&itdDateMonth=" + month + "&itdTimeHour=" + hour + "&anySigWhenPerfectNoOtherMatches=1" +
		"&locationServerActive=1&name_origin=" + origin + "&itdDateDay=" + day + "&type_origin=any" +
		"&name_destination=" + destination + "&itdTimeMinute=" + minute + "&Session=0&stateless=1" +
		"&SpEncId=0&itdDateYear=" + year
}
