package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/danackerson/bender-slackbot/structures"
	"github.com/danackerson/digitalocean/common"
	jsoniter "github.com/json-iterator/go"
	"github.com/nlopes/slack"
)

var rtm *slack.RTM
var joinAPIKey = os.Getenv("CTX_JOIN_API_KEY")
var vpnGateway = os.Getenv("CTX_VPNC_GATEWAY")
var githubRunID = os.Getenv("GITHUB_RUN_ID")

// Logger to give senseful settings
var Logger = log.New(os.Stdout, "", log.LstdFlags)

// VPNCountry as default connection
var VPNCountry = "NL"

// SlackReportChannel default reporting channel for bot crons
var SlackReportChannel = os.Getenv("CTX_SLACK_CHANNEL")

// SetRTM sets singleton
func SetRTM(rtmPassed *slack.RTM) {
	rtm = rtmPassed
}

// CheckCommand is now commented
func CheckCommand(api *slack.Client, slackMessage slack.Msg, command string) {
	args := strings.Fields(command)
	callingUserProfile, _ := api.GetUserInfo(slackMessage.User)
	params := slack.MsgOptionAsUser(true)

	if args[0] == "scpxl" {
		if len(args) > 1 {
			// strip '<>' off url
			downloadURL := strings.Trim(args[1], "<>")
			uri, err := url.ParseRequestURI(downloadURL)
			Logger.Printf("parsed %s from %s", uri.RequestURI(), downloadURL)
			if err != nil {
				api.PostMessage(slackMessage.Channel,
					slack.MsgOptionText(
						"Invalid URL for downloading! ("+err.Error()+
							")", true), params)
			} else {
				remoteClient := scpRemoteConnectionConfiguration(structures.AndroidRCC)
				if scpFileBetweenHosts(
					remoteClient,
					downloadURL,
					structures.AndroidRCC.HostPath) {
					api.PostMessage(slackMessage.Channel,
						slack.MsgOptionText(
							"Requested URL...", true), params)
				} else {
					api.PostMessage(slackMessage.Channel,
						slack.MsgOptionText(
							"Unable to download URL...", true), params)
				}
			}
		} else {
			api.PostMessage(slackMessage.Channel,
				slack.MsgOptionText("Please provide source file URL!", true), params)
		}
	} else if args[0] == "crypto" {
		response := checkEthereumValue() + "\n" + checkStellarLumensValue()
		api.PostMessage(slackMessage.Channel,
			slack.MsgOptionText(response, false), params)
	} else if args[0] == "pgp" {
		api.PostMessage(slackMessage.Channel,
			slack.MsgOptionText(pgpKeys(), false), params)
	} else if args[0] == "pi" {
		api.PostMessage(slackMessage.Channel,
			slack.MsgOptionText(raspberryPIChecks(), false), params)
	} else if args[0] == "yt" {
		if len(args) > 1 {
			// strip '<>' off url
			downloadURL := strings.Trim(args[1], "<>")
			uri, err := url.ParseRequestURI(downloadURL)
			Logger.Printf("parsed %s from %s", uri.RequestURI(), downloadURL)
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
	} else if args[0] == "wgu" {
		response := wireguardAction("up")
		rtm.SendMessage(rtm.NewOutgoingMessage(response, slackMessage.Channel))
	} else if args[0] == "wgd" {
		response := wireguardAction("down")
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
		if len(args) > 1 {
			searchString := strings.Join(args[1:], " ")
			searchStringURL := "/api?url=/q.php?q=" + url.QueryEscape(searchString)

			response = parseTorrents(searchProxy(searchStringURL))
		} else {
			response = parseTop100(searchProxy("/api?url=/precompiled/data_top100_207.json"))
		}

		api.PostMessage(slackMessage.Channel, slack.MsgOptionText(response, false), params)
	} else if args[0] == "vpns" {
		if len(args) > 1 {
			VPNCountry = strings.ToUpper(args[1])
		}
		response := VpnPiTunnelChecks(VPNCountry, true)
		rtm.SendMessage(rtm.NewOutgoingMessage(response, slackMessage.Channel))
	} else if args[0] == "vpnc" {
		if len(args) > 1 {
			vpnServerDomain := strings.ToLower(scrubParamOfHTTPMagicCrap(args[1]))
			// ensure vpnServerDomain has format e.g. DE-19
			var rxPat = regexp.MustCompile(`^[A-Za-z]{2}-[0-9]{2}`)
			if !rxPat.MatchString(vpnServerDomain) {
				rtm.SendMessage(
					rtm.NewOutgoingMessage(
						"Provide a validly formatted VPN server (hint: output from `vpns`)",
						slackMessage.Channel))
			} else {
				response := updateVpnPiTunnel(vpnServerDomain)
				rtm.SendMessage(
					rtm.NewOutgoingMessage(response, slackMessage.Channel))
			}
		} else {
			rtm.SendMessage(
				rtm.NewOutgoingMessage(
					"Please provide a new VPN server (hint: output from `vpns`)",
					slackMessage.Channel))
		}
	} else if args[0] == "version" {
		response := ":github: <https://github.com/ackersonde/bender-slackbot/actions/runs/" +
			githubRunID + "|" + githubRunID + ">"
		api.PostMessage(slackMessage.Channel, slack.MsgOptionText(response, false), params)
	} else if args[0] == "sw" {
		response := ":partly_sunny_rain: <https://darksky.net/forecast/48.3028,11.3591/ca24/en#week|7-day forecast Schwabhausen>"
		api.PostMessage(slackMessage.Channel, slack.MsgOptionText(response, false), params)
	} else if args[0] == "trans" || args[0] == "trand" || args[0] == "tranc" || args[0] == "tranp" {
		response := torrentCommand(args)
		api.PostMessage(slackMessage.Channel, slack.MsgOptionText(response, false), params)
	} else if args[0] == "mvv" {
		response := "<" + mvvRoute("Schwabhausen", "München, Hauptbahnhof") + "|Going in>"
		response += " | <" + mvvRoute("München, Hauptbahnhof", "Schwabhausen") + "|Going home>"

		response += "\n" + fetchAktuelles()

		api.PostMessage(slackMessage.Channel, slack.MsgOptionText(response, false), params)
	} else if args[0] == "www" {
		fritzBox := ":fritzbox: <https://fritz.ackerson.de/|fritz.box> | "
		fritzBox += ":traefik: <https://monitor.ackerson.de/dashboard/#/ | traefik> | "
		pi4 := ":k8s: <https://dash.ackerson.de/#/overview?namespace=default|k8s> | "
		pi4 += ":netdata: <http://pi4:19999/#menu_cpu_submenu_utilization;theme=slate;help=true|netdata>\n"
		pi4 += ":pihole: <http://hole.ackerson.de/admin/|pi.hole> | "
		vpnpi := ":transmission: <http://vpnpi:9091/transmission/web/|trans> | "
		vpnpi += ":plex: <http://vpnpi:32400/web/index.html#|plex> | "
		vpnpi += ":traefik: <https://api-wc-gcp.ackerson.de/dashboard/#/ | weechat>\n"

		response := fritzBox + pi4 + vpnpi
		api.PostMessage(slackMessage.Channel, slack.MsgOptionText(response, false), params)
	} else if args[0] == "help" {
		response :=
			":ethereum: `crypto`: Current cryptocurrency stats :lumens:\n" +
				":sleuth_or_spy: `pgp`: PGP keys\n" +
				":sun_behind_rain_cloud: `sw`: Schwabhausen weather\n" +
				":mvv: `mvv`: Status | Trip In | Trip Home\n" +
				":baseball: `bb <YYYY-MM-DD>`: show baseball games from given date (default yesterday)\n" +
				//":do_droplet: `do|dd <id>`: show|delete DigitalOcean droplet(s)\n" +
				":wireguard: `wg[s|u|d]`: [S]how status, [U]p or [D]own wireguard tunnel\n" +
				":protonvpn: `vpn[s|c]`: [S]how status of VPN on :raspberry_pi:, [C]hange VPN to best in given country or " + VPNCountry + "\n" +
				":pirate_bay: `torq <search term>`\n" +
				":transmission: `tran[c|p|s|d]`: [C]reate <URL>, [P]aused <URL>, [S]tatus, [D]elete <ID> torrents on :raspberry_pi:\n" +
				":movie_camera: `mv " + piPlexPath + "/torrents/<filename> [movies|tv/(<path>)]`\n" +
				":youtube: `yt <video url>`: Download Youtube video to Papa's handy\n" +
				":floppy_disk: `fsck`: show disk space on :raspberry_pi:\n" +
				":bar_chart: `pi`: Stats of various :raspberry_pi:s\n" +
				":github: `version`: Which build number is this Bender?\n" +
				":earth_americas: `www`: Show various internal links\n" +
				":copyright: `scpxl <URL>`: scp URL file to Pops4XL\n"
		api.PostMessage(slackMessage.Channel, slack.MsgOptionText(response, true), params)
	} else if callingUserProfile != nil {
		rtm.SendMessage(rtm.NewOutgoingMessage("whaddya say <@"+callingUserProfile.Name+">? Try `help` instead...",
			slackMessage.Channel))
	} else {
		Logger.Printf("No Command found: %s", slackMessage.Text)
	}
}

func fetchAktuelles() string {
	rndString := strconv.FormatInt(time.Now().UnixNano(), 10)
	url := "https://db-streckenagent.hafas.de/newsletter/gate?rnd=" + rndString

	// Goto https://www.s-bahn-muenchen.de/s_muenchen/view/service/aktuelle_betriebslage.shtml w/ DevTools enabled
	// inspect REQs like https://db-streckenagent.hafas.de/newsletter/gate?rnd=xyz
	// as I expect the post data below may change regularly :(
	var postData = []byte(`{"id":"ssww7rjiiqci9m88","ver":"1.25","lang":"deu","auth":{"type":"AID","aid":"da39a3ee5e6b4"},"client":{"id":"HAFAS","type":"WEB","name":"webapp","l":"vs_webapp"},"formatted":false,"svcReqL":[{"req":{"getChildren":true,"getParent":true,"maxNum":500,"himFltrL":[{"mode":"INC","type":"CH","value":"CUSTOM1"}],"sortL":["LMOD_DESC"]},"meth":"HimSearch","id":"1|1|"}]}`)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(postData))
	if err != nil {
		log.Printf("ERR prep request: %s", err.Error())
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("ERR make request: %s", err.Error())
	}
	defer resp.Body.Close()

	response, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("ERR read resp: %s", err.Error())
	}

	/* TEST response
	response := []byte(`
	{"ver":"1.25","lang":"deu","id":"ssww7rjiiqci9m88","err":"OK","graph":{"id":"standard","index":0},"subGraph":{"id":"global","index":0},"view":{"id":"standard","index":0,"type":"WGS84"},"svcResL":[{"id":"1|1|","meth":"HimSearch","err":"OK","res":{"common":{"locL":[{"lid":"A=1@O=München Ost@X=11604975@Y=48127437@U=80@L=8000262@","type":"S","name":"München Ost","icoX":0,"extId":"8000262","state":"F","crd":{"x":11604993,"y":48127302,"z":0},"pCls":447},{"lid":"A=1@O=München Donnersbergerbrücke@X=11536540@Y=48142620@U=80@L=8004128@","type":"S","name":"München Donnersbergerbrücke","icoX":1,"extId":"8004128","state":"F","crd":{"x":11537133,"y":48142683,"z":0},"pCls":56}],"prodL":[{"name":"S 1"},{"name":"S 6"},{"name":"S 7"},{"name":"S 8"},{"name":"S 2"}],"icoL":[{"res":"prod_ice","fg":{"r":255,"g":255,"b":255},"bg":{"r":40,"g":45,"b":55}},{"res":"prod_reg","fg":{"r":255,"g":255,"b":255},"bg":{"r":175,"g":180,"b":187}},{"res":"HIM1"}],"himMsgEdgeL":[{"icoCrd":{"x":11570757,"y":48135028}}],"himMsgCatL":[{"id":1}],"gTagL":["titleText","emailTitle","operationalSituationTitle","operationalSituation","email"]},"msgL":[{"hid":"RIS_HIM_FREETEXT_1080490","act":true,"head":"Bauarbeiten.","icoX":2,"prio":50,"fLocX":0,"tLocX":1,"prod":65535,"affProdRefL":[0,1,2,3,4],"src":99,"lModDate":"20200529","lModTime":"202744","sDate":"20200529","sTime":"223000","eDate":"20200601","eTime":"043000","sDaily":"000000","eDaily":"235900","comp":"Region Bayern","catRefL":[0],"pubChL":[{"name":"EMAIL","fDate":"20200529","fTime":"201500","tDate":"20200601","tTime":"043000"},{"name":"CUSTOM1","fDate":"20200529","fTime":"201500","tDate":"20200601","tTime":"043000"}],"edgeRefL":[0],"texts":[{"gTagXL":[0],"texts":[{"text":"Bauarbeiten."}]},{"gTagXL":[1,2],"texts":[{"text":"Stammstrecke: Bauarbeiten von Freitag, 29. Mai, 22.30 Uhr bis Montag, 01. Juni 2020, 4.30 Uhr zwischen München Ost und München-Pasing"}]},{"gTagXL":[3,4],"texts":[{"text":"Wegen Bauarbeiten zur 2.Stammstrecke kommt es von Freitag, 29. Mai (22:30 Uhr) durchgehend bis Montag, 1. Juni 2020 (4:30 Uhr) zwischen München Ost und München-Pasing zu Fahrplanänderungen mit Umleitungen und Haltausfällen auf fast allen S-Bahn-Linien. <br><br>Zwischen München Ost und München-Pasing verkehren nur die Linien S 6 und S 7 regulär durch die Stammstrecke.<br>Zwischen München-Ost und Hackerbrücke besteht am Samstag, jeweils von 9 bis 1 Uhr und am Sonntag, jeweils von 11 bis 21 Uhr ein Pendelverkehr im 20-Minuten-Takt.<br><br>Weitere Informationen, sowie die Fahrpläne der einzelnen Linien finden Sie unter https://t1p.de/94jj"}]}]}]}}]}`)
	*/

	svcResL := jsoniter.Get(response, "svcResL", 0).ToString()
	res := jsoniter.Get([]byte(svcResL), "res").ToString()
	common := jsoniter.Get([]byte(res), "common").ToString()
	if common == "{}" {
		return "Aktuell liegen uns keine Meldungen vor."
	}

	aktuell := ""
	lines := strings.ToLower(jsoniter.Get([]byte(common), "prodL", '*').ToString())
	if lines != "" {
		effectedTrains := []map[string]string{}
		effectedTrainLogos := ""

		err2 := json.Unmarshal([]byte(lines), &effectedTrains)
		if err2 != nil {
			log.Printf("ERR parsing trains: %s", err2.Error())
		} else {
			for _, train := range effectedTrains {
				logo := strings.Replace(train["name"], " ", "", 1)
				effectedTrainLogos += fmt.Sprintf(":mvv_" + logo + ": ")
			}
			effectedTrainLogos += "\n"
			aktuell = fmt.Sprintf("%s\n", effectedTrainLogos)
		}
	}

	msgL := jsoniter.Get([]byte(res), "msgL", 0).ToString()
	titleTexts := jsoniter.Get([]byte(msgL), "texts", 1).ToString()
	gTagXL1 := jsoniter.Get([]byte(titleTexts), "texts", 0).ToString()
	titleText := jsoniter.Get([]byte(gTagXL1), "text").ToString()

	subjectTexts := jsoniter.Get([]byte(msgL), "texts", 2).ToString()
	gTagXL2 := jsoniter.Get([]byte(subjectTexts), "texts", 0).ToString()
	subjectText := jsoniter.Get([]byte(gTagXL2), "text").ToString()
	subjectText = strings.ReplaceAll(subjectText, "<br>", "\n")

	aktuell += fmt.Sprintf("%s\n\n%s\n\n", titleText, subjectText)

	lModDate := jsoniter.Get([]byte(msgL), "lModDate").ToString()
	lModTime := jsoniter.Get([]byte(msgL), "lModTime").ToString()
	if lModDate != "" && lModTime != "" {
		lastUpdate, _ := time.Parse("20060102150405", lModDate+lModTime)
		aktuell += fmt.Sprintf("_update von %s_", lastUpdate.Format("02-Jan-2006 15:04"))
	}

	return aktuell
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
