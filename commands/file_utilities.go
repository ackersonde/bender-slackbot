package commands

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/ackersonde/bender-slackbot/structures"
	"github.com/slack-go/slack"
)

var mediaPath = "/mnt/usb4TB/DLNA"

// CheckServerDiskSpace now exported
func CheckServerDiskSpace(path string) string {
	if path != "" {
		path = scrubParamOfHTTPMagicCrap(path)
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
				response += "\n" + line
			}
		}
	}
	response += "\n==========================\n"

	return ":raspberry_pi: *SD Card Disk Usage* `pi4`\n" + response
}

// CheckServerDiskSpaceCron called by scheduler
func CheckServerDiskSpaceCron(path string) {
	api.PostMessage(SlackReportChannel, slack.MsgOptionText(
		CheckServerDiskSpace(path), false), slack.MsgOptionAsUser(true))
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

// Check disk space of important devices - only report if > 95%
func CheckDiskSpace() {
	response := ""
	response += checkDiskSpace("pi4", "/dev/mmcblk0p2")
	response += checkDiskSpace("vpnpi", "/dev/mmcblk0p2")
	response += checkDiskSpace("vpnpi", "/dev/sda1")
	response += checkDiskSpace("hetzner", "/")
	response += checkDiskSpace("hetzner", "/mnt/hetzner_disk")

	if response != "" {
		api.PostMessage(SlackReportChannel, slack.MsgOptionText(
			response, false), slack.MsgOptionAsUser(true))
	}
}

func checkDiskSpace(server string, mount string) string {
	response := ""
	cmdPrefix := ""
	sshConfig := structures.PI4RemoteConnectConfig

	if server == "hetzner" {
		cmdPrefix = "ssh vault "
	} else if server == "vpnpi" {
		sshConfig = structures.VPNPIRemoteConnectConfig
	}

	cmd := fmt.Sprintf("%sdf %s | sed 1d | awk '{ print $5 }'", cmdPrefix, mount)
	Logger.Printf("checkDiskSpace: %s", cmd)
	remoteResult := executeRemoteCmd(cmd, sshConfig)

	if remoteResult.Stdout == "" && remoteResult.Stderr != "" {
		response = remoteResult.Stderr
	} else {
		response = remoteResult.Stdout
		// take the resulting string and get it's numeric value e.g. "29%" => 29
		i, err := strconv.Atoi(strings.TrimRight(response, "%"))
		if err != nil {
			response = fmt.Sprintf("%s@%s: unable to parse %s: %s\n", server, mount, response, err)
		} else if i >= 15 {
			response = fmt.Sprintf("%s@%s: disk used *%d%%* :rotating_light:\n", server, mount, i)
		} else { // disk space is < 95% -> OK
			Logger.Printf("%s@%s: disk used %d%%\n", server, mount, i)
			response = ""
		}
	}

	return response
}

// CheckMediaDiskSpaceCron called by scheduler
func CheckMediaDiskSpaceCron(path string) {
	api.PostMessage(SlackReportChannel, slack.MsgOptionText(
		CheckMediaDiskSpace(path), false), slack.MsgOptionAsUser(true))
}

func CheckHetznerSpace(path string) string {
	response := ""

	cmd := fmt.Sprintf("ssh vault df %s", path)
	Logger.Printf("cmd: %s", cmd)
	remoteResult := executeRemoteCmd(cmd, structures.PI4RemoteConnectConfig)

	if remoteResult.Stdout == "" && remoteResult.Stderr != "" {
		response = remoteResult.Stderr
	} else {
		response = remoteResult.Stdout
	}

	return response
}

func CheckDigitalOceanSpace(path string) string {
	response := ""
	out, err := exec.Command("df", "-h", syncthing).Output()
	if err != nil {
		Logger.Printf("ERR: %s", err.Error())
	}

	response = ":do_droplet: DO *Disk Usage* `root@" + syncthing +
		"`\n" + string(out)

	return response
}

// MoveTorrentFile now exported
func MoveTorrentFile(sourceFile string, destinationDir string) {
	params := slack.MsgOptionAsUser(true)
	response := ""

	cmd := fmt.Sprintf("mv %s/%s %s/%s", mediaPath, sourceFile, mediaPath, destinationDir)
	remoteResult := executeRemoteCmd(cmd, structures.VPNPIRemoteConnectConfig)

	if remoteResult.Stdout == "" && remoteResult.Stderr != "" {
		response = fmt.Sprint(remoteResult.Stderr)
		response = ":x: ERR: `" + cmd + "` => " + response
	} else {
		response += fmt.Sprintf("moved: %s to %s\n", sourceFile, destinationDir)
	}

	api.PostMessage(SlackReportChannel, slack.MsgOptionText(response, false), params)
}
