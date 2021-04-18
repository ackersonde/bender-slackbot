package commands

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/ackersonde/bender-slackbot/structures"
	"github.com/bramvdbogaerde/go-scp"
	"github.com/slack-go/slack"
)

var mediaPath = "/mnt/usb4TB/DLNA"

// CheckServerDiskSpace now exported
func CheckServerDiskSpace(path string) string {
	if path != "" {
		path = strings.TrimSuffix(path, "/")
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
	}

	response := ""
	cmd := fmt.Sprintf("/bin/df -h %s", path)
	remoteResult := executeRemoteCmd(cmd, structures.PI4RemoteConnectConfig)
	if remoteResult.Stdout == "" && remoteResult.Stderr != "" {
		response = remoteResult.Stderr
	} else {
		for _, line := range strings.Split(
			strings.TrimSuffix(remoteResult.Stdout, "\n"), "\n") {
			if strings.Contains(line, "loop") ||
				strings.Contains(line, "tmpfs") {
				continue
			} else {
				response += line
			}
		}
	}

	return ":raspberry_pi: *SD Card Disk Usage* `pi4`\n" + response
}

// CheckMediaDiskSpace now exported
func CheckMediaDiskSpace(path string) string {
	if path != "" {
		path = strings.TrimSuffix(path, "/")
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
	}

	response := ""
	cmd := fmt.Sprintf("du -hd 1 %s | sort -k 1", mediaPath+path)
	if !strings.HasSuffix(path, "/*") {
		cmd += " | sed '1d'"
	}
	Logger.Printf("cmd: %s", cmd)
	remoteResult := executeRemoteCmd(cmd, structures.VPNPIRemoteConnectConfig)

	if remoteResult.Stdout == "" && remoteResult.Stderr != "" {
		response = remoteResult.Stderr
	} else {
		response = remoteResult.Stdout
	}

	response = ":jelly: USB *Disk Usage* `vpnpi@" + mediaPath + path +
		"`\n" + response

	cmd = fmt.Sprintf("/bin/df -h %s /", mediaPath+path)
	remoteResult = executeRemoteCmd(cmd, structures.VPNPIRemoteConnectConfig)

	if remoteResult.Stdout == "" && remoteResult.Stderr != "" {
		response += "\n" + remoteResult.Stderr
	} else {
		response += "\n" + remoteResult.Stdout
	}
	response += "\n==========================\n"

	return response
}

// MoveTorrentFile now exported
func MoveTorrentFile(sourceFile string, destinationDir string) {
	params := slack.MsgOptionAsUser(true)
	response := ""

	cmd := fmt.Sprintf("mv %s/%s %s/%s", mediaPath, sourceFile, mediaPath, destinationDir)
	remoteResult := executeRemoteCmd(cmd, structures.VPNPIRemoteConnectConfig)

	if remoteResult.Stdout == "" && remoteResult.Stderr != "" {
		response = fmt.Sprintf(fmt.Sprint(remoteResult.Stderr) + ": " + string(remoteResult.Stdout))
		response = ":x: ERR: `" + cmd + "` => " + response
	} else {
		response += fmt.Sprintf("moved: %s to %s\n", sourceFile, destinationDir)
	}

	api.PostMessage(SlackReportChannel, slack.MsgOptionText(response, false), params)
}

func scpFileBetweenHosts(remoteClient scp.Client, sourceURI string, hostPath string) bool {
	fetchURL, _ := url.Parse(sourceURI)
	destination := ""
	success := false

	if strings.Contains(fetchURL.Hostname(), "youtu.be") ||
		strings.Contains(fetchURL.Hostname(), "youtube.com") {
		fetchURL, destination = findVideoOnYoutube(fetchURL)
	} else {
		// get filename from URL end "/<filename.ext>"
		path := fetchURL.Path
		segments := strings.Split(path, "/")

		destination = strings.ReplaceAll(segments[len(segments)-1], " ", "_")
	}

	response, err := http.Get(fetchURL.String())
	if err != nil {
		Logger.Printf(err.Error())
		return success
	}

	// Close http connection after copying
	defer response.Body.Close()
	defer remoteClient.Close()

	destination = strings.TrimLeft(destination, "-.") // remove leading '.'s & '-'s
	Logger.Printf("scp %s %s@%s\n", sourceURI, remoteClient.Host, hostPath+destination)

	err = remoteClient.CopyFile(response.Body, hostPath+destination, "0644")
	if err != nil {
		Logger.Printf("Error while copying file %s", err)
	} else {
		success = true
	}

	return success
}
