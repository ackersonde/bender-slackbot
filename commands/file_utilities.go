package commands

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/ackersonde/bender-slackbot/structures"
	"github.com/bramvdbogaerde/go-scp"
	"github.com/slack-go/slack"
)

var piPlexPath = "/mnt/usb4TB/DLNA"

// CheckServerDiskSpace now exported
func CheckServerDiskSpace(path string) {
	if path != "" {
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
	api.PostMessage(SlackReportChannel, slack.MsgOptionText(response, false),
		slack.MsgOptionAsUser(true))
}

// CheckMediaDiskSpace now exported
func CheckMediaDiskSpace(path string) {
	if path != "" {
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
	Logger.Printf("cmd: %s", cmd)
	remoteResult := executeRemoteCmd(cmd, structures.VPNPIRemoteConnectConfig)

	if remoteResult.Stdout == "" && remoteResult.Stderr != "" {
		response = remoteResult.Stderr
	} else {
		response = remoteResult.Stdout
	}

	response = ":plex: USB *Disk Usage* `vpnpi@" + piPlexPath + path +
		"`\n" + response + "\n"

	cmd = fmt.Sprintf("/bin/df -h %s /", piPlexPath+path)
	remoteResult = executeRemoteCmd(cmd, structures.VPNPIRemoteConnectConfig)

	if remoteResult.Stdout == "" && remoteResult.Stderr != "" {
		response = response + "\n" + remoteResult.Stderr
	} else {
		response = response + "\n" + remoteResult.Stdout
	}
	response += "\n=============================\n"

	api.PostMessage(SlackReportChannel, slack.MsgOptionText(response, false),
		slack.MsgOptionAsUser(true))
}

type basePlexRefreshCmdString struct {
	HostName string
	Section  string
	Token    string
}

func (i basePlexRefreshCmdString) String() string {
	return fmt.Sprintf("http://%s:32400/library/sections/%s/refresh?X-Plex-Token=%s",
		i.HostName, i.Section, i.Token)
}

// MoveTorrentFile now exported
func MoveTorrentFile(sourceFile string, destinationDir string) {
	params := slack.MsgOptionAsUser(true)
	response := ""

	cmd := fmt.Sprintf("mv %s/%s %s/%s", piPlexPath, sourceFile, piPlexPath, destinationDir)
	remoteResult := executeRemoteCmd(cmd, structures.VPNPIRemoteConnectConfig)

	if remoteResult.Stdout == "" && remoteResult.Stderr != "" {
		response = fmt.Sprintf(fmt.Sprint(remoteResult.Stderr) + ": " + string(remoteResult.Stdout))
		response = ":x: ERR: `" + cmd + "` => " + response
	} else {
		response += fmt.Sprintf("moved: %s to %s\n", sourceFile, destinationDir)
		librarySection := "1"
		plexToken := os.Getenv("CTX_PLEX_TOKEN")
		if strings.HasPrefix(destinationDir, "tv") {
			librarySection = "2"
		}

		refreshPlexTorrents := fmt.Sprintf(
			"curl %s",
			basePlexRefreshCmdString{
				HostName: structures.VPNPIRemoteConnectConfig.HostName,
				Section:  "1", Token: plexToken})
		refreshPlexSection := fmt.Sprintf(
			"curl %s",
			basePlexRefreshCmdString{
				HostName: structures.VPNPIRemoteConnectConfig.HostName,
				Section:  librarySection, Token: plexToken})
		refreshCmd := fmt.Sprintf("%s && %s", refreshPlexTorrents, refreshPlexSection)

		out, err := exec.Command("ash", "-c", refreshCmd).Output()
		if err != nil {
			response += fmt.Sprintf(fmt.Sprint(err) + ": " + string(out))
			response = ":x: ERR: `" + refreshCmd + "` => " + response
		} else {
			response += fmt.Sprintf(
				"refreshed <http://%s:32400/web/index.html|Plex library %s>\n",
				structures.VPNPIRemoteConnectConfig.HostName, librarySection)
		}
	}

	api.PostMessage(SlackReportChannel, slack.MsgOptionText(response, false), params)
}

func scpFileBetweenHosts(remoteClient scp.Client, sourceURI string, hostPath string) bool {
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
