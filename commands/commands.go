package commands

import (
	"bufio"
	"encoding/hex"
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

	"github.com/danackerson/digitalocean/common"
	humanize "github.com/dustin/go-humanize"
	"github.com/elgs/gojq"
	"github.com/nlopes/slack"
	"golang.org/x/crypto/scrypt"
)

var raspberryPIIP = os.Getenv("raspberryPIIP")
var rtm *slack.RTM
var piSDCardPath = "/home/pi/torrents/"
var piUSBMountPoint = "/mnt/usb_1"
var piUSBMountPath = piUSBMountPoint + "/DLNA/torrents/"
var routerIP = "192.168.1.1"

var spacesKey = os.Getenv("SPACES_KEY")
var spacesSecret = os.Getenv("SPACES_SECRET")
var spacesNamePublic = os.Getenv("SPACES_NAME_PUBLIC")
var joinAPIKey = os.Getenv("joinAPIKey")

var circleCIDoAlgoURL = "https://circleci.com/api/v1.1/project/github/danackerson/do-algo"
var circleCITokenParam = "?circle-token=" + os.Getenv("circleAPIToken")

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
				api.PostMessage(slackMessage.Channel, slack.MsgOptionText("Invalid URL for downloading! ("+err.Error()+")", true), params)
			} else {
				downloadYoutubeVideo(uri.String())
				api.PostMessage(slackMessage.Channel, slack.MsgOptionText("Requested YouTube video...", true), params)
			}
		} else {
			api.PostMessage(slackMessage.Channel, slack.MsgOptionText("Please provide YouTube video URL!", true), params)
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
	} else if args[0] == "algo" {
		response := ListDODroplets(true)
		region := "fra1"
		if len(args) > 1 {
			region = args[1]
		}

		if strings.Contains(response, "york.shire") {
			response = findAndReturnVPNConfigs(response, region)
			api.PostMessage(slackMessage.Channel, slack.MsgOptionText(response, false), params)
		} else {
			building, buildNum, _ := circleCIDoAlgoBuildingAndBuildNums(region)
			if !building {
				buildsURL := circleCIDoAlgoURL + circleCITokenParam
				data := url.Values{}
				data.Set("build_parameters[REGION]", region)
				data.Set("build_parameters[CIRCLE_JOB]", "deploy")
				buildsParser := getJSONFromRequestURL(buildsURL, "POST", data.Encode())

				buildNumParse, _ := buildsParser.Query("build_num")
				buildNum = strconv.FormatFloat(buildNumParse.(float64), 'f', -1, 64)
			}
			response = ":circleci: <https://circleci.com/gh/danackerson/do-algo/" + buildNum + "|do-algo Build " + buildNum + " @ " + region + ">"
			api.PostMessage(slackMessage.Channel, slack.MsgOptionText(response, false), params)

		}
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
		if runningFritzboxTunnel() {
			response := ""

			if len(args) > 1 {
				path := strings.Join(args[1:], " ")
				response += CheckPiDiskSpace(path)
			} else {
				response += CheckPiDiskSpace("")
			}

			// grab listing from FritzBox NAS
			ftpListingCmd := "curl -s ftp://ftpuser:abc123@192.168.178.1/backup/DLNA/torrents/ | awk '{print $5\"\t\"$9}'"
			ftpListDetails := RemoteCmd{Host: raspberryPIIP, HostKey: piHostKey, Username: os.Getenv("piUser"), Password: os.Getenv("piPass"), Cmd: ftpListingCmd}
			remoteResult := executeRemoteCmd(ftpListDetails)

			diskUsage := getUSBDiskUsageOnFritzBox(remoteResult.stdout)
			response += "\n\n:wifi: USB Disk ~/torrents on :fritzbox:\n" + diskUsage

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
					MoveTorrentFile(api, path)
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
			searchString = strings.TrimPrefix(searchString, "torq ")
			_, response = SearchFor(searchString, Category(cat))
		} else {
			_, response = SearchFor("", Category(cat))
		}
		api.PostMessage(slackMessage.Channel, slack.MsgOptionText(response, false), params)
	} else if args[0] == "ovpn" {
		response := RaspberryPIPrivateTunnelChecks(true)
		rtm.SendMessage(rtm.NewOutgoingMessage(response, slackMessage.Channel))
	} else if args[0] == "sw" {
		response := ":partly_sunny_rain: <https://darksky.net/forecast/48.3028,11.3591/ca24/en#week|7-day forecast Schwabhausen>"
		api.PostMessage(slackMessage.Channel, slack.MsgOptionText(response, false), params)
	} else if args[0] == "vpnc" {
		result := vpnTunnelCmds("/usr/sbin/vpnc-connect", "fritzbox")
		rtm.SendMessage(rtm.NewOutgoingMessage(result, slackMessage.Channel))
	} else if args[0] == "vpnd" {
		result := vpnTunnelCmds("/usr/sbin/vpnc-disconnect")
		rtm.SendMessage(rtm.NewOutgoingMessage(result, slackMessage.Channel))
	} else if args[0] == "vpns" {
		result := vpnTunnelCmds("status")
		rtm.SendMessage(rtm.NewOutgoingMessage(result, slackMessage.Channel))
	} else if args[0] == "trans" || args[0] == "trand" || args[0] == "tranc" || args[0] == "tranp" {
		if runningFritzboxTunnel() {
			response := torrentCommand(args)
			api.PostMessage(slackMessage.Channel, slack.MsgOptionText(response, false), params)
		}
	} else if args[0] == "mvv" {
		response := "<https://img.srv2.de/customer/sbahnMuenchen/newsticker/newsticker.html|Aktuelles>"
		response += " | <" + mvvRoute("Schwabhausen", "München, Hauptbahnhof") + "|Going in>"
		response += " | <" + mvvRoute("München, Hauptbahnhof", "Schwabhausen") + "|Going home>"

		api.PostMessage(slackMessage.Channel, slack.MsgOptionText(response, false), params)
	} else if args[0] == "help" {
		response :=
			":sun_behind_rain_cloud: `sw`: Schwabhausen weather\n" +
				":metro: `mvv`: Status | Trip In | Trip Home\n" +
				":do_droplet: `do|dd <id>`: show|delete DigitalOcean droplet(s)\n" +
				":algovpn: `algo (nyc1|tor1|lon1|ams3|...)`: show|launch AlgoVPN droplet on :do_droplet: (in given region - default FRA1)\n" +
				":closed_lock_with_key: `vpn[c|s|d]`: [C]onnect, [S]tatus, [D]rop VPN tunnel to Fritz!Box\n" +
				":openvpn: `ovpn`: show status of OVPN.se on :raspberry_pi:\n" +
				":pirate_bay: `torq <search term>`\n" +
				":transmission: `tran[c|p|s|d]`: [C]reate <URL>, [P]aused <URL>, [S]tatus, [D]elete <ID> torrents on :raspberry_pi:\n" +
				":recycle: `rm(|mv) <filename>` from :raspberry_pi: (to `" + piUSBMountPath + "`)\n" +
				":floppy_disk: `fsck`: show disk space on :raspberry_pi:\n" +
				":baseball: `bb <YYYY-MM-DD>`: show baseball games from given date (default yesterday)\n" +
				":youtube: `yt <video url>`: Download Youtube video to Papa's handy\n"
		api.PostMessage(slackMessage.Channel, slack.MsgOptionText(response, true), params)
	} else if callingUserProfile != nil {
		rtm.SendMessage(rtm.NewOutgoingMessage("whaddya say <@"+callingUserProfile.Name+">? Try `help` instead...",
			slackMessage.Channel))
	} else {
		log.Printf("No Command found: %s", slackMessage.Text)
	}
}

func circleCIDoAlgoBuildingAndBuildNums(region string) (bool, string, string) {
	lastSuccessBuildNum := "-1"
	currentBuildNum := "-1"
	currentlyBuilding := true

	buildsURL := circleCIDoAlgoURL + circleCITokenParam
	buildsParser := getJSONFromRequestURL(buildsURL, "GET", "")
	array, _ := buildsParser.QueryToArray(".")
	for i := 0; i < len(array); i++ {
		statusStr, _ := buildsParser.Query("[" + strconv.Itoa(i) + "].status")

		if i == 0 {
			log.Println("current Do-Algo build status: " + statusStr.(string))
			currentlyBuilding = !isFinishedStatus(statusStr.(string))
			buildNumParse, _ := buildsParser.Query("[" + strconv.Itoa(i) + "].build_num")
			currentBuildNum = strconv.FormatFloat(buildNumParse.(float64), 'f', -1, 64)
		}

		if statusStr.(string) == "success" || statusStr.(string) == "fixed" {
			buildNumParse, _ := buildsParser.Query("[" + strconv.Itoa(i) + "].build_num")
			lastSuccessBuildNum = strconv.FormatFloat(buildNumParse.(float64), 'f', -1, 64)
			break
		}
	}

	return currentlyBuilding, currentBuildNum, lastSuccessBuildNum
}

func isFinishedStatus(status string) bool {
	switch status {
	case
		"canceled",
		"success",
		"fixed",
		"failed":
		return true
	}
	return false
}

func getUSBDiskUsageOnFritzBox(ftpDirectories string) string {
	result := ""
	directoryHint := "4096\t"

	scanner := bufio.NewScanner(strings.NewReader(ftpDirectories))
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), directoryHint) {
			directory := strings.SplitAfter(scanner.Text(), directoryHint)[1]
			totalSize := scanDirectory(directory + "/")
			result += fmt.Sprintf("%s\t%s\n", humanize.Bytes(totalSize), directory)
		} else {
			fileInfo := strings.Split(scanner.Text(), "\t")
			fileSize, err := strconv.ParseUint(fileInfo[0], 10, 64)
			if err != nil {
				fmt.Println(err.Error())
			}
			result += fmt.Sprintf("%s\t%s\n", humanize.Bytes(fileSize), fileInfo[1])
		}
	}
	if err := scanner.Err(); err != nil {
		log.Println(err)
	}

	return result
}

func scanDirectory(directoryName string) uint64 {
	var size uint64

	ftpListingCmd := `curl -s ftp://ftpuser:abc123@192.168.178.1/backup/DLNA/torrents/` + directoryName + ` | awk '{print $5"\t"$9}'`
	fmt.Println("Running `" + ftpListingCmd + "`")
	ftpListDetails := RemoteCmd{Host: raspberryPIIP, HostKey: piHostKey, Username: os.Getenv("piUser"), Password: os.Getenv("piPass"), Cmd: ftpListingCmd}
	remoteResult := executeRemoteCmd(ftpListDetails)

	scanner := bufio.NewScanner(strings.NewReader(remoteResult.stdout))
	for scanner.Scan() {
		fileInfo := strings.Split(scanner.Text(), "\t")
		fileSize, err := strconv.ParseUint(fileInfo[0], 10, 64)
		if err != nil {
			fmt.Println(err.Error())
		}
		size += fileSize
	}
	if err := scanner.Err(); err != nil {
		log.Println(err)
	}

	return size
}

func getJSONFromRequestURL(url string, requestType string, encodedData string) *gojq.JQ {
	req, _ := http.NewRequest(requestType, url, strings.NewReader(encodedData))
	if requestType == "POST" && encodedData != "" {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("failed to call "+url+": ", err)
	} else {
		log.Println("successfully called: " + url)
	}
	defer resp.Body.Close()

	contentBytes, _ := ioutil.ReadAll(resp.Body)
	contentParser, _ := gojq.NewStringQuery(string(contentBytes))

	return contentParser
}

func waitAndRetrieveLogs(buildURL string, index int) string {
	outputURL := "N/A"

	for notReady := true; notReady; notReady = (outputURL == "N/A") {
		buildParser := getJSONFromRequestURL(buildURL, "GET", "")
		actionsParser, errOutput := buildParser.Query("steps.[" + strconv.Itoa(index) + "].actions.[0].output_url")

		if errOutput != nil {
			log.Println("waitAndRetrieveLogs: " + errOutput.Error())
			time.Sleep(5000 * time.Millisecond)
		} else {
			outputURL = actionsParser.(string)
		}
	}

	return outputURL
}

func findAndReturnVPNConfigs(doServers string, region string) string {
	passAlgoVPN := "No successful AlgoVPN deployments found."
	links := ""

	building, _, lastSuccessBuildNum := circleCIDoAlgoBuildingAndBuildNums(region)

	if building {
		// sleep 10 secs and check again
		time.Sleep(10 * time.Second)
		building, _, lastSuccessBuildNum = circleCIDoAlgoBuildingAndBuildNums(region)
	}

	if !building && lastSuccessBuildNum != "-1" {
		// now get build details for this buildNum
		var outputURL string
		buildURL := circleCIDoAlgoURL + "/" + lastSuccessBuildNum + circleCITokenParam
		buildParser := getJSONFromRequestURL(buildURL, "GET", "")
		for i := 0; i < 9; i++ {
			stepName, _ := buildParser.Query("steps.[" + strconv.Itoa(i) + "].name")
			if stepName == "deploy to Digital Ocean Droplet & launch VPN" {
				outputURL = waitAndRetrieveLogs(buildURL, i)
				break
			}
		}

		// get the log output for this step and parse out IP address and SSH password
		outputParser := getJSONFromRequestURL(outputURL, "GET", "")
		message, error := outputParser.QueryToString("[0].message")
		if error != nil {
			log.Printf("QueryToString ERR: %s", error.Error())
		}
		log.Printf("outputParser val: %s", message)

		checkPassString, _ := regexp.Compile(`The p12 and SSH keys password for new users is (?:[0-9a-zA-Z_@]{8})`)
		passAlgoVPN = string(checkPassString.Find([]byte(message)))

		ipv4 := getIPv4Address(doServers)

		// lets encrypt the filenames on disk
		doPersonalAccessToken := os.Getenv("digitalOceanToken")
		salt := []byte(ipv4 + ":" + doPersonalAccessToken)
		desktopConfigFileHashed, _ := scrypt.Key([]byte("dan.mobileconfig"), salt, 16384, 8, 1, 32)
		desktopConfigFileString := hex.EncodeToString(desktopConfigFileHashed) + ".mobileconfig"
		fmt.Println(desktopConfigFileString)

		mobileConfigFileHashed, _ := scrypt.Key([]byte("dan.conf"), salt, 16384, 8, 1, 32)
		mobileConfigFileString := hex.EncodeToString(mobileConfigFileHashed) + ".conf"
		fmt.Println(mobileConfigFileString)

		localMobileConfigFilePath := "/algo_vpn/" + ipv4 + "/wireguard/dan.conf"
		localDesktopConfigFilePath := "/algo_vpn/" + ipv4 + "/dan.mobileconfig"

		remoteMobileConfigURL := "/.recycle/" + mobileConfigFileString
		remoteDesktopConfigURL := "/.recycle/" + desktopConfigFileString

		err := common.CopyFileToDOSpaces(spacesNamePublic, localDesktopConfigFilePath, remoteDesktopConfigURL, -1)
		if err != nil {
			log.Printf("Unable to upload %s to Spaces %s", localDesktopConfigFilePath, err.Error())
		} else {
			err := common.CopyFileToDOSpaces(spacesNamePublic, localMobileConfigFilePath, remoteMobileConfigURL, -1)
			if err != nil {
				log.Printf("Unable to upload %s to Spaces %s", localMobileConfigFilePath, err.Error())
			} else {
				joinStatus := "*Import* VPN profile"

				icon := "http://www.setaram.com/wp-content/themes/setaram/library/images/lock.png"
				smallIcon := "http://www.setaram.com/wp-content/themes/setaram/library/images/lock.png"

				// 2. Change below Join Push alert to S3 bucket URL
				sendPayloadToJoinAPI(remoteMobileConfigURL, "dan.conf", icon, smallIcon)
				digitalOceanSpacesURL := spacesNamePublic + ".ams3.digitaloceanspaces.com"
				links = ":link: <https://" + digitalOceanSpacesURL + remoteMobileConfigURL + "|dan_" + ipv4 + ".conf> (" + joinStatus + ")\n"
				links += ":link: <https://" + digitalOceanSpacesURL + remoteDesktopConfigURL + "|dan.mobileconfig> (dbl click on Mac)\n"
			}
		}
	}

	return ":algovpn: " + passAlgoVPN + "\n" + links
}

func getIPv4Address(serverList string) string {
	var ipV4 []byte

	parts := strings.Split(serverList, "\n")
	for i := range parts {
		// FORMAT => ":do_droplet: <addr|name> (IPv4) [ID: DO_ID]"
		if strings.Contains(parts[i], "york.shire") {
			reIPv4, _ := regexp.Compile(`(?:[0-9]{1,3}\.){3}[0-9]{1,3}`)
			ipV4 = reIPv4.Find([]byte(parts[i]))
			break
		}
	}

	return string(ipV4)
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

func downloadYoutubeVideo(origURL string) bool {
	resp, err := http.Get("https://ackerson.de/bb_download?gameURL=" + origURL)
	if err != nil {
		log.Printf("ERR: downloading YTube video: %s", err.Error())
	}
	if resp.StatusCode == 200 {
		return true
	}

	return false
}
