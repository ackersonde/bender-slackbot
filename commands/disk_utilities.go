package commands

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/nlopes/slack"
)

var raspberryPIIP = "raspberrypi.fritz.box"
var pi4 = "pi4.fritz.box"

var piTorrentsPath = "/home/pi/torrents"
var piPlexPath = "/mnt/usb4TB/DLNA"

// CheckTorrentsDiskSpace now exported
func CheckTorrentsDiskSpace(path string) string {
	userCall := true
	if path == "---" {
		path = ""
		userCall = false
	} else {
		path = strings.ReplaceAll(path, "/", "")
	}

	cmd := "du -h " + piTorrentsPath + path + "/*"

	response := ""
	details := RemoteCmd{Host: raspberryPIIP, HostKey: piHostKey, Username: "pi", Cmd: cmd}
	remoteResult := executeRemoteCmd(details)

	if remoteResult.stdout == "" && remoteResult.stderr != "" {
		response = remoteResult.stderr
	} else {
		response = remoteResult.stdout
	}
	response = ":raspberry_pi: *SD Card Disk Usage* @ `" + piTorrentsPath + path + "`\n" + response

	cmd = "df -h /root/"
	details = RemoteCmd{Host: raspberryPIIP, HostKey: piHostKey, Username: "pi", Cmd: cmd}
	remoteResultDF := executeRemoteCmd(details)
	response += "\n\n" + remoteResultDF.stdout

	if !userCall {
		customEvent := slack.RTMEvent{Type: "CheckPiDiskSpace", Data: response}
		rtm.IncomingEvents <- customEvent
	}

	return response
}

// DeleteTorrentFile now exported
func DeleteTorrentFile(filename string) string {
	var response string

	if filename == "*" || filename == "" || strings.Contains(filename, "../") {
		response = "Please enter an existing filename - try `fsck`"
	} else {
		path := piTorrentsPath + filename

		var deleteCmd string
		cmd := "test -d \"" + path + "\" && echo 'Yes'"
		details := RemoteCmd{Host: raspberryPIIP, HostKey: piHostKey, Username: "pi", Cmd: cmd}

		remoteResult := executeRemoteCmd(details)
		if strings.HasPrefix(remoteResult.stdout, "Yes") {
			deleteCmd = "rm -Rf \"" + path + "\""
		} else {
			deleteCmd = "rm \"" + path + "\""
		}

		details = RemoteCmd{Host: raspberryPIIP, HostKey: piHostKey, Username: "pi", Cmd: deleteCmd}

		remoteResultDelete := executeRemoteCmd(details)
		tunnelIdleSince = time.Now()
		if remoteResultDelete.stderr != "" {
			response = remoteResultDelete.stderr
		} else {
			response = remoteResultDelete.stdout
		}
	}

	return response
}

// MoveTorrentFile now exported
func MoveTorrentFile(api *slack.Client, filename string) {
	if filename == "" || strings.Contains(filename, "../") || strings.HasPrefix(filename, "/") {
		rtm.IncomingEvents <- slack.RTMEvent{Type: "MoveTorrent", Data: "Please enter an existing filename - try `fsck`"}
	} else {
		// detox filenames => http://detox.sourceforge.net/ | https://linux.die.net/man/1/detox
		renameCmd := "cd " + piTorrentsPath + "; rm *.log; detox -r *"
		renameDetails := RemoteCmd{Host: raspberryPIIP, HostKey: piHostKey, Username: "pi", Cmd: renameCmd}
		executeRemoteCmd(renameDetails)

		moveCmd := "cd " + piTorrentsPath + "; find . -type f -exec curl -g --ftp-create-dirs -u ftpuser:abc123 -T \"{}\" \"ftp://192.168.178.1/backup/DLNA/torrents/{}\" \\; > ftp.log 2>&1"
		log.Println(moveCmd)
		go func() {
			details := RemoteCmd{Host: raspberryPIIP, HostKey: piHostKey, Username: "pi", Cmd: moveCmd}
			var result string
			remoteResult := executeRemoteCmd(details)
			log.Printf("%v:%v", details, remoteResult)

			result = "Successfully moved `" + filename + "` to `" + piPlexPath + "`"
			rtm.IncomingEvents <- slack.RTMEvent{Type: "MoveTorrent", Data: result}

			cleanPITorrentsCmd := "cd " + piTorrentsPath + "; rm -Rf *;"
			details = RemoteCmd{Host: raspberryPIIP, HostKey: piHostKey,
				Username: "pi",
				Cmd:      cleanPITorrentsCmd}
			remoteResult = executeRemoteCmd(details)
		}()

		params := slack.MsgOptionAsUser(true)
		api.PostMessage(SlackReportChannel, slack.MsgOptionText("running `"+moveCmd+"`", true), params)
		//reportMoveProgress(api)
	}
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
			progressDetails := RemoteCmd{Host: raspberryPIIP, HostKey: piHostKey, Username: "pi", Cmd: progressCmd}

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