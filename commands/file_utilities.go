package commands

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/bramvdbogaerde/go-scp"
	"github.com/nlopes/slack"
)

var raspberryPIIP = "vpnpi.fritz.box"
var pi4 = "pi4.fritz.box"
var raspi3 = "192.168.178.25"

var piTorrentsPath = "/home/pi/torrents"
var piPlexPath = "/mnt/usb4TB/DLNA"

// CheckServerDiskSpace now exported
func CheckServerDiskSpace(path string) string {
	userCall := true
	if path == "---" {
		path = ""
		userCall = false
	} else if path != "" {
		path = strings.TrimSuffix(path, "/")
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
	}

	response := ""
	out2, err2 := exec.Command("/bin/df", "-h", "/").Output()
	if err2 != nil {
		response += err2.Error()
	} else {
		response += string(out2)
	}

	response = ":k8s: *SD Card Disk Usage* `pi4`\n" + response
	if !userCall {
		customEvent := slack.RTMEvent{Type: "CheckPiDiskSpace", Data: response}
		rtm.IncomingEvents <- customEvent
	}

	return response
}

// CheckMediaDiskSpace now exported
func CheckMediaDiskSpace(path string) string {
	userCall := true
	if path == "---" {
		path = ""
		userCall = false
	} else if path != "" {
		path = strings.TrimSuffix(path, "/")
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
	}

	response := ""
	cmd := fmt.Sprintf("du -hd 1 %s | sort -k 1", piPlexPath+path)
	if !strings.HasSuffix(path, "/*") {
		cmd += " | sed '1d'"
	}
	log.Printf("cmd: %s", cmd)
	details := RemoteCmd{Host: raspberryPIIP, Cmd: cmd}
	remoteResult := executeRemoteCmd(details, remoteConnectionConfiguration(vpnPIHostKey, "pi"))

	if remoteResult.stdout == "" && remoteResult.stderr != "" {
		response = remoteResult.stderr
	} else {
		response = remoteResult.stdout
	}

	response = ":plex: USB *Disk Usage* `vpnpi@" + piPlexPath + path +
		"`\n" + response + "\n"

	cmd = fmt.Sprintf("/bin/df -h %s /", piPlexPath+path)
	details = RemoteCmd{Host: raspberryPIIP, Cmd: cmd}
	remoteResult = executeRemoteCmd(details, remoteConnectionConfiguration(vpnPIHostKey, "pi"))

	if remoteResult.stdout == "" && remoteResult.stderr != "" {
		response = response + "\n" + remoteResult.stderr
	} else {
		response = response + "\n" + remoteResult.stdout
	}
	response += "\n=============================\n"

	if !userCall {
		customEvent := slack.RTMEvent{Type: "CheckPiDiskSpace", Data: response}
		rtm.IncomingEvents <- customEvent
	}

	return response
}

// MoveTorrentFile now exported
func MoveTorrentFile(api *slack.Client, sourceFile string, destinationDir string) {
	params := slack.MsgOptionAsUser(true)
	response := ""

	moveCmd := fmt.Sprintf("mv %s/%s %s/%s", piPlexPath, sourceFile, piPlexPath, destinationDir)
	details := RemoteCmd{Host: raspberryPIIP, Cmd: moveCmd}
	remoteResult := executeRemoteCmd(details, remoteConnectionConfiguration(vpnPIHostKey, "pi"))

	if remoteResult.stdout == "" && remoteResult.stderr != "" {
		response = fmt.Sprintf(fmt.Sprint(remoteResult.stderr) + ": " + string(remoteResult.stdout))
		response = ":x: ERR: `" + moveCmd + "` => " + response
	} else {
		response += fmt.Sprintf("moved: %s to %s\n", sourceFile, destinationDir)
		librarySection := "1"
		plexToken := os.Getenv("PLEX_TOKEN")
		if strings.HasPrefix(destinationDir, "tv") {
			librarySection = "2"
		}
		refreshCmd := fmt.Sprintf(
			"curl http://"+raspberryPIIP+":32400/library/sections/3/refresh?X-Plex-Token=%s && "+
				"curl http://"+raspberryPIIP+":32400/library/sections/%s/refresh?X-Plex-Token=%s",
			plexToken, librarySection, plexToken)
		out, err := exec.Command("ash", "-c", refreshCmd).Output()
		if err != nil {
			response += fmt.Sprintf(fmt.Sprint(err) + ": " + string(out))
			response = ":x: ERR: `" + refreshCmd + "` => " + response
		} else {
			response += fmt.Sprintf(
				"refreshed <http://vpnpi:32400/web/index.html|Plex library %s>\n",
				librarySection)
		}
	}

	api.PostMessage(SlackReportChannel, slack.MsgOptionText(response, false), params)

	//reportMoveProgress(api)
}

func ScpFileBetweenHosts(remoteClient scp.Client, sourceURI string, hostPath string) bool {
	fetchURL, err := url.Parse(sourceURI)
	destination := ""
	success := false

	if strings.Contains(fetchURL.Hostname(), "youtu.be") ||
		strings.Contains(fetchURL.Hostname(), "youtube.com") {
		fetchURL, destination = findVideoOnYoutube(fetchURL)
	} else {
		// get filename from URL end "/<filename.ext>"
		path := fetchURL.Path
		segments := strings.Split(path, "/")

		destination = segments[len(segments)-1]
	}

	response, err := http.Get(fetchURL.String())
	if err != nil {
		fmt.Println(err)
		return success
	}

	// Close http connection after copying
	defer response.Body.Close()
	defer remoteClient.Close()

	destination = strings.TrimLeft(destination, "-.") // remove leading '.'s & '-'s
	log.Printf("scp %s %s@%s\n", sourceURI, remoteClient.Host, hostPath+destination)

	err = remoteClient.CopyFile(response.Body, hostPath+destination, "0644")
	if err != nil {
		fmt.Println("Error while copying file ", err)
	} else {
		success = true
	}

	return success
}

func reportMoveProgress(api *slack.Client) {
	historyParams := new(slack.HistoryParameters)
	historyParams.Latest = ""
	historyParams.Count = 1
	historyParams.Inclusive = true
	lastMsgID := ""
	msgHistory, _ := api.GetChannelHistory(SlackReportChannel, *historyParams)
	for _, msg := range msgHistory.Messages {
		lastMsgID = msg.Timestamp
	}

	remoteResults := make(chan RemoteResult, 1)
	timeout := time.After(10 * time.Second)
	notDone := true

	for notDone {
		go func() {
			progressCmd := "progress"
			progressDetails := RemoteCmd{Host: raspberryPIIP, Cmd: progressCmd}

			remoteResults <- executeRemoteCmd(progressDetails, remoteConnectionConfiguration(vpnPIHostKey, "pi"))
		}()

		// reset tunnel idle time as user may want to see progress of move
		tunnelIdleSince = time.Now()

		select {
		case res := <-remoteResults:
			// update msg with progress: https://api.slack.com/methods/chat.update
			// so there aren't 385 msgs with 2% 2% 3% ...
			api.UpdateMessage(SlackReportChannel, lastMsgID, slack.MsgOptionText(res.stdout, true))
			if strings.Contains(res.stderr, "No command currently running") {
				notDone = false
			} else {
				time.Sleep(time.Second * 5)
			}
		case <-timeout:
			fmt.Println("Timed out!")
		}
	}
}
