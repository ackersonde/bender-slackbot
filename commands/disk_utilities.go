package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/nlopes/slack"
)

var raspberryPIIP = "192.168.178.59"
var pi4 = "pi4.fritz.box"

var piTorrentsPath = "/home/pi/torrents"
var piPlexPath = "/mnt/usb4TB/DLNA"

// CheckTorrentsDiskSpace now exported
func CheckTorrentsDiskSpace(path string) string {
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

	cmd := "du -bh " + piTorrentsPath + path + "/*"

	response := ""
	details := RemoteCmd{Host: raspberryPIIP, Cmd: cmd}
	remoteResult := executeRemoteCmd(details)

	if remoteResult.stdout == "" && remoteResult.stderr != "" {
		response = remoteResult.stderr
	} else {
		response = remoteResult.stdout
	}
	response = ":raspberry_pi: *SD Card Disk Usage* `vpnpi@" + piTorrentsPath +
		path + "`\n" + response

	cmd = "df -h /root/"
	details = RemoteCmd{Host: raspberryPIIP, Cmd: cmd}
	remoteResultDF := executeRemoteCmd(details)
	response += "\n\n" + remoteResultDF.stdout + "\n"

	if !userCall {
		customEvent := slack.RTMEvent{Type: "CheckPiDiskSpace", Data: response}
		rtm.IncomingEvents <- customEvent
	}

	return response
}

// CheckPlexDiskSpace now exported
func CheckPlexDiskSpace(path string) string {
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
	out, err := exec.Command("sh", "-c", cmd).Output() // ash -> sh ?
	if err != nil {
		response = fmt.Sprintf(fmt.Sprint(err) + ": " + string(out))
	} else {
		response = string(strings.Replace(string(out), piPlexPath+"/", "", -1))
	}

	response = ":plex: *Pi4 Card Disk Usage* `pi4@" + piPlexPath + path +
		"`\n" + response + "\n"

	out2, err2 := exec.Command("/bin/df", "-h", "/mnt/usb4TB").Output()
	if err2 != nil {
		response += err2.Error()
	} else {
		response += string(out2)
	}
	response += "\n==============================\n"

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
	out, err := exec.Command("ash", "-c", moveCmd).Output()
	if err != nil {
		response = fmt.Sprintf(fmt.Sprint(err) + ": " + string(out))
		response = ":x: ERR: `" + moveCmd + "` => " + response
	} else {
		response += fmt.Sprintf("moved: %s to %s\n", sourceFile, destinationDir)
		librarySection := "1"
		plexToken := os.Getenv("PLEX_TOKEN")
		if strings.HasPrefix(destinationDir, "tv") {
			librarySection = "5"
		}
		refreshCmd := fmt.Sprintf(
			"curl http://pi4:32400/library/sections/%s/refresh?X-Plex-Token=%s",
			librarySection, plexToken)
		out, err := exec.Command("ash", "-c", refreshCmd).Output()
		if err != nil {
			response += fmt.Sprintf(fmt.Sprint(err) + ": " + string(out))
			response = ":x: ERR: `" + refreshCmd + "` => " + response
		} else {
			response += fmt.Sprintf(
				"refreshed <http://pi4:32400/web/index.html|Plex library %s>\n",
				librarySection)
		}
	}

	api.PostMessage(SlackReportChannel, slack.MsgOptionText(response, true), params)

	//reportMoveProgress(api)
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

			remoteResults <- executeRemoteCmd(progressDetails)
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
