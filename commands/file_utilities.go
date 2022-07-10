package commands

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

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

// Check disk space of important devices
func CheckDiskSpace() string {
	response := ""
	response += checkDiskSpaceOfServer("pi4", "/dev/mmcblk0p2")
	response += checkDiskSpaceOfServer("vpnpi", "/dev/mmcblk0p2")
	response += checkDiskSpaceOfServer("vpnpi", "/dev/sda1")
	response += checkDiskSpaceOfServer("hetzner", "/")
	response += checkDiskSpaceOfServer("hetzner", "/mnt/hetzner_disk")

	// only report back if something is amiss
	if response != "" {
		api.PostMessage(SlackReportChannel, slack.MsgOptionText(
			response, false), slack.MsgOptionAsUser(true))
	}

	return response
}

func checkDiskSpaceOfServer(server string, mount string) string {
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
		// if error || >= 95% report an error
		i, err := strconv.Atoi(strings.TrimRight(response, "%\n"))
		if err != nil {
			response = fmt.Sprintf("%s@%s: unable to parse %s: %s\n", server, mount, response, err)
		} else if i >= 92 { // only report if > 92%
			response = fmt.Sprintf("%s@%s: disk used *%d%%* :rotating_light:\n", server, mount, i)
		} else { // disk space is < 92% -> OK
			Logger.Printf("%s@%s: disk used %d%%\n", server, mount, i)
			response = "" // don't bother me
		}
	}

	return response
}

// CheckMediaDiskSpaceCron called by scheduler
func CheckMediaDiskSpaceCron(path string) {
	api.PostMessage(SlackReportChannel, slack.MsgOptionText(
		CheckMediaDiskSpace(path), false), slack.MsgOptionAsUser(true))
}

func CheckBackups() string {
	response := ""

	now := time.Now()
	//lastMonth := now.AddDate(0, -1, 0)
	calcDate := now.Format("2006/01/")

	response += checkBackupDirectory("vpnpi", "/mnt/usb4TB/backups/vault-secrets/")
	response += checkBackupDirectory("vpnpi", "/mnt/usb4TB/backups/photos/originals/"+calcDate)
	response += checkBackupDirectory("hetzner", "/mnt/hetzner_disk/backups/photos/"+calcDate)

	return response
}

func checkBackupDirectory(server string, path string) string {
	response, cmdPrefix, cmdSuffix := "", "", ""
	sshConfig := structures.VPNPIRemoteConnectConfig

	if server == "hetzner" {
		cmdPrefix = "ssh vault '"
		cmdSuffix = "'"
		sshConfig = structures.PI4RemoteConnectConfig
	}

	cmd := fmt.Sprintf("%sls -l %s | wc -l && du -sh %s%s", cmdPrefix, path, path, cmdSuffix)
	// e.g.
	// 108
	// 495M	/mnt/hetzner_disk/backups/photos/2022/05/

	Logger.Printf("checkBackups: %s", cmd)

	remoteResult := executeRemoteCmd(cmd, sshConfig)
	if remoteResult.Stdout == "" && remoteResult.Stderr != "" {
		response = remoteResult.Stderr
	} else {
		scanner := bufio.NewScanner(strings.NewReader(remoteResult.Stdout))
		row := 0
		for scanner.Scan() {
			text := scanner.Text()
			if row == 0 {
				files, err := strconv.Atoi(text)
				if err != nil || files < 2 {
					response += "No files in the backup dir `" + path + "`"
					break
				}
			} else if strings.HasPrefix(text, "4.0K") {
				response += "`" + path + "` exists, but currently empty"
			}
			row++
		}
	}

	return response
}

func CheckHetznerSpace(path string, showHeader bool) string {
	response := ""
	suffix := ""

	if !showHeader {
		suffix = " | sed 1d"
	}

	cmd := fmt.Sprintf("ssh vault df -h %s%s", path, suffix)
	Logger.Printf("CheckHetznerSpace: %s", cmd)
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

	response = ":do_droplet: *DO Disk Usage* `root@" + syncthing +
		"`\n" + string(out)

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
		response = fmt.Sprint(remoteResult.Stderr)
		response = ":x: ERR: `" + cmd + "` => " + response
	} else {
		response += fmt.Sprintf("moved: %s to %s\n", sourceFile, destinationDir)
	}

	api.PostMessage(SlackReportChannel, slack.MsgOptionText(response, false), params)
}
