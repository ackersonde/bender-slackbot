package commands

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ackersonde/bender-slackbot/structures"
	"github.com/ackersonde/digitaloceans/common"
	"github.com/ackersonde/hetzner/hetznercloud"
	jsoniter "github.com/json-iterator/go"
	"github.com/sethvargo/go-password/password"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

var api *slack.Client
var githubRunID = os.Getenv("GITHUB_RUN_ID")

// Logger to give senseful settings
var Logger = log.New(os.Stdout, "", log.LstdFlags)

// VPNCountry as default connection
var VPNCountry = "NL"

// Syncthing directory
var syncthing = "/app/sync/"

// SlackReportChannel default reporting channel for bot crons
var SlackReportChannel = os.Getenv("CTX_SLACK_CHANNEL")

// SetAPI sets singleton
func SetAPI(apiPassed *slack.Client) {
	api = apiPassed
}

// CheckCommand is now commented
func CheckCommand(event *slackevents.MessageEvent, user *slack.User, command string) {
	args := strings.Fields(command)
	params := slack.MsgOptionAsUser(true)

	if args[0] == "crypto" {
		response := checkEthereumValue() + "\n" + checkStellarLumensValue()
		api.PostMessage(event.Channel,
			slack.MsgOptionText(response, false), params)
	} else if args[0] == "pgp" {
		api.PostMessage(event.Channel,
			slack.MsgOptionText(pgpKeys(), false), params)
	} else if args[0] == "pi" {
		api.PostMessage(event.Channel,
			slack.MsgOptionText(raspberryPIChecks(), false), params)
	} else if args[0] == "wf" {
		if len(args) < 2 {
			args = append(args, "STATE") // empty cmd shows wifi status
		}
		api.PostMessage(event.Channel,
			slack.MsgOptionText(WifiAction(args[1]), false), params)
	} else if args[0] == "yt" {
		if len(args) > 1 {
			// strip '<>' off url
			downloadURL := strings.Trim(args[1], "<>")
			uri, err := url.ParseRequestURI(downloadURL)
			Logger.Printf("parsed %s from %s", uri.RequestURI(), downloadURL)
			if err != nil {
				api.PostMessage(event.Channel,
					slack.MsgOptionText(
						"Invalid URL for downloading! ("+err.Error()+")", true), params)
			} else {
				_, err := exec.Command("/usr/bin/youtube-dl", uri.String(),
					"-o", syncthing+"%(title)s.%(ext)s").Output()
				if err == nil {
					api.PostMessage(event.Channel,
						slack.MsgOptionText(
							"Requested YouTube video. Check Syncthing in a few minutes...", true), params)
				} else {
					api.PostMessage(event.Channel,
						slack.MsgOptionText(
							"Unable to download YouTube video..."+err.Error(), true), params)
				}
			}
		} else {
			api.PostMessage(event.Channel,
				slack.MsgOptionText("Please provide YouTube video URL!", true), params)
		}
	} else if args[0] == "bb" {
		result := ""
		dateString := ""

		if len(args) > 1 {
			gameDate, err := time.Parse("2006-01-02", args[1])
			dateString = gameDate.Format("2006/month_01/day_02")

			if err != nil {
				result = "Couldn't figure out date '" + args[1] + "'. Try `help`"
				api.PostMessage(event.Channel, slack.MsgOptionText(result, false), params)
				return
			}
		}
		result = ShowBBGames(dateString)
		api.PostMessage(event.Channel, slack.MsgOptionText(result, false), params)
	} else if args[0] == "logs" {
		result := "Unable to query docker..."
		if len(args) > 1 {
			result = dockerInfo(args[1])
		} else {
			result = dockerInfo("")
		}
		api.PostMessage(event.Channel, slack.MsgOptionText(result, false), params)
	} else if args[0] == "pass" {
		usage := "Usage: pass <64 10 10 false true>\nwhere default params are <chars:64 digits:10 symbols:10 upper&lower:false repeatChars:true>.\nOrder counts! If you only need to change `upper&lower`, you *must* enter preceding params."
		response := ""
		var err error

		chars := 64
		digits := 10
		symbols := 10

		switch len(args) {
		case 2:
			chars, _ = strconv.Atoi(args[1])

			if chars < 20 {
				digits = chars / 2
				symbols = digits
			}
			response, err = password.Generate(chars, digits, symbols, false, false)
			if err != nil {
				response = err.Error() + "\n" + usage
			}
		case 3:
			chars, _ := strconv.Atoi(args[1])
			digits, _ := strconv.Atoi(args[2])

			if chars-digits > 10 {
				symbols = chars - digits
			}
			response, err = password.Generate(chars, digits, symbols, false, false)
			if err != nil {
				response = err.Error() + "\n" + usage
			}
		case 4:
			chars, _ := strconv.Atoi(args[1])
			digits, _ := strconv.Atoi(args[2])
			symbols, _ := strconv.Atoi(args[3])

			response, err = password.Generate(chars, digits, symbols, false, false)
			if err != nil {
				response = err.Error() + "\n" + usage
			}
		case 5:
			chars, _ := strconv.Atoi(args[1])
			digits, _ := strconv.Atoi(args[2])
			symbols, _ := strconv.Atoi(args[3])
			upAndLow, _ := strconv.ParseBool(args[4])

			response, err = password.Generate(chars, digits, symbols, upAndLow, false)
			if err != nil {
				response = err.Error() + "\n" + usage
			}
		case 6:
			chars, _ := strconv.Atoi(args[1])
			digits, _ := strconv.Atoi(args[2])
			symbols, _ := strconv.Atoi(args[3])
			upAndLow, _ := strconv.ParseBool(args[4])
			repeatChars, _ := strconv.ParseBool(args[5])

			response, err = password.Generate(chars, digits, symbols, upAndLow, !repeatChars)
			if err != nil {
				response = err.Error() + "\n" + usage
			}
		default:
			response, err = password.Generate(64, 10, 10, false, false)
			if err != nil {
				response = err.Error() + "\n" + usage
			}
		}

		api.PostMessage(event.Channel, slack.MsgOptionText(response, false), params)
	} else if args[0] == "vlg" {
		// TODO: check vault-traefik logs via:
		// `awk '{print $1}' traefik/logs/access.log|sort|uniq -c|sort|grep -v '<HOME_IPV6_PREFIX>'|grep -v '172.17.0'|tail -n 10`
		// where <HOME_IPV6_PREFIX> is first 4 octals of fetchHomeIPv6Prefix()
		// e.g. 2a00:432a:40:9154::/62 => 2a00:432a:40:9154
		// show last 10
	} else if args[0] == "vfa" {
		response := "Usage: vfa <(get) keyname | put keyname secret>"

		totpEngineName := "totp"
		if event.User != "U092UC9EW" {
			totpEngineName = "liuda"
		}

		if len(args) == 1 || (len(args) == 2 && args[1] == "get") {
			response = fmt.Sprintf("%v\n", listTOTPKeysForEngine(totpEngineName))
		} else if args[1] != "put" && args[1] != "update" {
			keyname := args[1]
			if args[1] == "get" {
				keyname = args[2]
			}
			response = fmt.Sprintf("%s\n", readTOTPCodeForKey(totpEngineName, keyname))
		} else if args[1] == "put" {
			response = fmt.Sprintf("%s\n", putTOTPKeyForEngine(totpEngineName, args[2], args[3]))
		} else if args[1] == "update" {
			response = fmt.Sprintf("%s\n", updateTOTPRoleCIDRs("totp-mgmt",
				fetchHomeIPv6Prefix()+","+args[2]))
		}

		api.PostMessage(event.Channel, slack.MsgOptionText(response, false), params)
	} else if args[0] == "htz" {
		response := "No servers at :htz_server:."
		servers := hetznercloud.ListAllServers()
		if len(servers) > 0 {
			response = "Found following server(s) at :htz_server::\n"
		}

		for _, server := range servers {
			serverInfoURL := fmt.Sprintf("https://console.hetzner.cloud/projects/1200165/servers/%d/overview", server.ID)
			serverIPv6 := server.PublicNet.IPv6.IP.String()
			if strings.HasSuffix(serverIPv6, "::") {
				serverIPv6 += "1"
			}

			response += fmt.Sprintf("ID %d: <%s|%s> [%s] @ %s => %s\n",
				server.ID, serverInfoURL, server.Name, serverIPv6,
				server.Created.Format("2006-01-02 15:04"), server.Status)
		}

		remoteResult := executeRemoteCmd("ssh vault 'uptime;uname -a'", structures.PI4RemoteConnectConfig)
		response += remoteResult.Stdout

		api.PostMessage(event.Channel, slack.MsgOptionText(response, false), params)
	} else if args[0] == "htzd" {
		if len(args) > 1 {
			serverID, err := strconv.Atoi(args[1])
			if err != nil {
				api.PostMessage(event.Channel, slack.MsgOptionText("Invalid integer value for ID!", true), params)
			} else {
				result := hetznercloud.DeleteServer(serverID)
				api.PostMessage(event.Channel, slack.MsgOptionText(result, true), params)
			}
		} else {
			api.PostMessage(event.Channel, slack.MsgOptionText("Please provide Droplet ID from `do` cmd!", true), params)
		}
	} else if args[0] == "do" {
		response := ListDODroplets()
		api.PostMessage(event.Channel, slack.MsgOptionText(response, false), params)
	} else if args[0] == "dd" {
		if len(args) > 1 {
			number, err := strconv.Atoi(args[1])
			if err != nil {
				api.PostMessage(event.Channel, slack.MsgOptionText("Invalid integer value for ID!", true), params)
			} else {
				result := common.DeleteDODroplet(number)
				api.PostMessage(event.Channel, slack.MsgOptionText(result, true), params)
			}
		} else {
			api.PostMessage(event.Channel, slack.MsgOptionText("Please provide Droplet ID from `do` cmd!", true), params)
		}
	} else if args[0] == "fsck" {
		response := ""
		if len(args) > 1 {
			path := strings.Join(args[1:], " ")
			response += CheckMediaDiskSpace(path)
			response += CheckServerDiskSpace("")
		} else {
			response += CheckMediaDiskSpace("")
			response += CheckServerDiskSpace("")
		}
		response += CheckDigitalOceanSpace("")
		api.PostMessage(event.Channel, slack.MsgOptionText(response, true), params)
	} else if args[0] == "mv" {
		if len(args) == 3 &&
			(strings.HasPrefix(args[2], "movies") ||
				strings.HasPrefix(args[2], "tv")) {
			sourceFile := scrubParamOfHTTPMagicCrap(args[1])
			destinationDir := args[2]
			if strings.Contains(destinationDir, "..") || strings.HasPrefix(destinationDir, "/") {
				msg := fmt.Sprintln("Please prefix destination w/ either `[movies|tv]`")
				api.PostMessage(event.Channel, slack.MsgOptionText(msg, true), params)
			} else if strings.Contains(sourceFile, "..") || strings.HasPrefix(sourceFile, "/") {
				msg := fmt.Sprintf("Please specify file to move relative to `%s/torrents/`\n", mediaPath)
				api.PostMessage(event.Channel, slack.MsgOptionText(msg, true), params)
			} else {
				MoveTorrentFile(sourceFile, destinationDir)
			}
		} else {
			msg := "Please provide a src file and destination [e.g. `movies` or `tv`]"
			api.PostMessage(event.Channel, slack.MsgOptionText(msg, true), params)
		}
	} else if args[0] == "torq" {
		var response string
		if len(args) > 1 {
			searchString := strings.Join(args[1:], " ")
			searchStringURL := "/q.php?q=" + url.QueryEscape(searchString)

			response = parseTorrents(searchProxy(searchStringURL))
		} else {
			response = parseTop100(searchProxy("/precompiled/data_top100_207.json"))
		}

		api.PostMessage(event.Channel, slack.MsgOptionText(response, false), params)
	} else if args[0] == "vpns" {
		if len(args) > 1 {
			VPNCountry = strings.ToUpper(args[1])
		}
		response := VpnPiTunnelChecks(VPNCountry)
		api.PostMessage(event.Channel, slack.MsgOptionText(response, false), params)
	} else if args[0] == "vpnc" {
		response := "Please provide a new VPN server (hint: output from `vpns`)"
		if len(args) > 1 {
			vpnServerDomain := strings.ToLower(scrubParamOfHTTPMagicCrap(args[1]))
			// ensure vpnServerDomain has format e.g. DE-19
			var rxPat = regexp.MustCompile(`^(lxc-)?[A-Za-z]{2}-[0-9]{2}`)
			if !rxPat.MatchString(vpnServerDomain) {
				response = "Provide a validly formatted VPN server (hint: output from `vpns`)"

			} else {
				response = updateVpnPiTunnel(vpnServerDomain)
			}
		}
		api.PostMessage(event.Channel, slack.MsgOptionText(response, false), params)
	} else if args[0] == "version" {
		fingerprint := getDeployFingerprint("/root/.ssh/id_ed25519-cert.pub")
		response := ":github: <https://github.com/ackersonde/bender-slackbot/actions/runs/" +
			githubRunID + "|" + githubRunID + "> using :key: " + fingerprint
		api.PostMessage(event.Channel, slack.MsgOptionText(response, false), params)
	} else if args[0] == "rw" {
		response := ":partly_sunny_rain: <https://openweathermap.org/city/2860447|8d forecast Oberhatzkofen>"
		api.PostMessage(event.Channel, slack.MsgOptionText(response, false), params)
	} else if args[0] == "trans" || args[0] == "trand" || args[0] == "tranc" || args[0] == "tranp" {
		response := torrentCommand(args)
		api.PostMessage(event.Channel, slack.MsgOptionText(response, false), params)
	} else if args[0] == "mvv" {
		response := "<" + mvvRoute("Schwabhausen", "München, Hauptbahnhof") + "|Going in>"
		response += " | <" + mvvRoute("München, Hauptbahnhof", "Schwabhausen") + "|Going home>"

		response += "\n" + fetchAktuelles()

		api.PostMessage(event.Channel, slack.MsgOptionText(response, false), params)
	} else if args[0] == "www" {
		digitalOcean := ":do_droplet:: <https://monitor.ackerson.de/dashboard/#/|:traefik:> | <https://sync.ackerson.de|:syncthing:> | <https://ackerson.de|:homepage:>\n"
		hetzner := ":htz_server:: <https://mv.ackerson.de/dashboard/#/|:traefik:> | <https://vault.ackerson.de/ui/|:vault:>\n"

		fritzBox := ":house:: <https://fritz.ackerson.de/|:fritzbox:> | <https://freedns.afraid.org/dynamic/v2/|:afraid:>\n"
		pi4 := ":raspberry_pi:: <https://ht.ackerson.de/dashboard/#/|:traefik:> | <https://homesync.ackerson.de|:syncthing:> | <https://photos.ackerson.de/|:photoprism:> | <http://192.168.178.27:8200|:vault:>\n"
		vpnpi := ":protonvpn:: <http://vpnpi.fritz.box:9091/transmission/web/|:transmission:> | <http://vpnpi:8096/web/index.html#!/home.html|:jelly:>\n"

		response := digitalOcean + hetzner + fritzBox + pi4 + vpnpi
		api.PostMessage(event.Channel, slack.MsgOptionText(response, false), params)
	} else if args[0] == "key" {
		response := getBendersCurrentSSHCert()
		api.PostMessage(event.Channel, slack.MsgOptionText(response, false), params)
	} else if args[0] == "security" {
		response := checkFirewallRules(true)
		api.PostMessage(event.Channel, slack.MsgOptionText(response, false), params)
	} else if args[0] == "help" {
		response :=
			":ethereum: `crypto`: Current cryptocurrency stats :lumens:\n" +
				":sleuth_or_spy: `pgp`: PGP keys\n" +
				":vault: `vfa <get (keyname)| put keyname secret>`: Vault Factor Auth (TOTP)\n" +
				":key: `pass (chars:64 digits:10 symbols:10 upper&lower:false repeatChars:true)`: Generate password with params\n" +
				":sun_behind_rain_cloud: `rw`: Oberhatzkofen weather\n" +
				":mvv: `mvv`: Status | Trip In | Trip Home\n" +
				":baseball: `bb <YYYY-MM-DD>`: show baseball games from given date (default yesterday)\n" +
				":do_droplet: `do|dd <id>`: show|delete DigitalOcean droplet(s)\n" +
				":htz_server: `htz|htzd <id>`: show|delete Hetzner server(s)\n" +
				//":wireguard: `wg[s|u|d]`: [S]how status, [U]p or [D]own wireguard tunnel\n" +
				":wifi: `wf [0|1|s]`: turn home wifi [0]ff, [1]n or [-default-s]tatus\n" +
				":protonvpn: `vpn[s|c]`: [S]how status of VPN on :raspberry_pi:, [C]hange VPN to best in given country or " + VPNCountry + "\n" +
				":pirate_bay: `torq <search term>`\n" +
				":transmission: `tran[c|p|s|d]`: [C]reate <URL>, [P]aused <URL>, [S]tatus, [D]elete <ID> torrents on :raspberry_pi:\n" +
				":movie_camera: `mv " + mediaPath + "/torrents/<filename> [movies|tv/(<path>)]`\n" +
				":youtube: `yt <video url>`: Download Youtube video to Papa's handy\n" +
				":floppy_disk: `fsck`: show disk space on :raspberry_pi:\n" +
				":bar_chart: `pi`: Stats of various :raspberry_pi:s\n" +
				":github: `version`: Which build/deploy is this Bender bot?\n" +
				":earth_americas: `www`: Show various internal links\n" +
				":copyright: `scpxl <URL>`: scp URL file to Pops4XL\n" +
				":closed_lock_with_key: `security`: overview of SSH key(s) and UFW rules\n" +
				":whale2: `logs <container>`: last 100 lines of docker logs from <container> on ackerson.de\n"
		api.PostMessage(event.Channel, slack.MsgOptionText(response, true), params)
	} else {
		response := "Whaddya say <@" + user.Profile.DisplayName + ">? Try `help` instead"
		api.PostMessage(event.Channel, slack.MsgOptionText(response, false), params)
	}
}

func getBendersCurrentSSHCert() string {
	response := ""
	out, err := exec.Command("ssh-keygen", "-L", "-f", "/root/.ssh/id_ed25519-cert.pub").Output()
	if err != nil {
		response += err.Error()
	} else {
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			text := strings.Trim(scanner.Text(), " ")
			if strings.HasPrefix(text, "Serial:") {
				response = text
				continue
			} else if strings.HasPrefix(text, "Valid:") {
				valid := text
				// Valid: from 2021-02-02T13:44:00 to 2021-03-09T13:45:01
				re := regexp.MustCompile(`Valid: from (?P<start>.*) to (?P<expire>.*)`)
				matches := re.FindAllStringSubmatch(text, -1)
				names := re.SubexpNames()

				m := map[string]string{}
				if len(matches) > 0 {
					for i, n := range matches[0] {
						m[names[i]] = n
					}
					if len(m) > 1 {
						expiry, err := time.Parse("2006-01-02T15:04:05", m["expire"])
						if err != nil {
							Logger.Printf("Unable to parse expiry date: %s", m["expire"])
						} else {
							today := time.Now()
							if expiry.Before(today) {
								valid += "\n" + ":rotating_light: Cert is expired! Please check `/var/log/gen_new_deploy_keys.log` and possibly rerun `/home/ubuntu/my-ca/gen_new_deploy_keys.sh` on pi4." +
									"\nOr just <https://github.com/ackersonde/bender-slackbot/actions|redeploy bender> ..."
							} else {
								daysValid := expiry.Sub(today).Hours() / 24
								valid += "\nSSH Certificate valid for " + strconv.FormatFloat(daysValid, 'f', 0, 64) + " days"
							}
						}
					} else {
						Logger.Printf("Unable to parse validity: %s", valid)
					}
				} else {
					Logger.Printf("ERR: PUB CERT invalid date: %s", valid)
				}

				response += "\n" + valid
				break
			}
		}
	}

	return response
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
