package commands

import (
	"fmt"
	"strings"

	"github.com/ackersonde/bender-slackbot/structures"
)

var transmissionSettingsPath = "/root/.config/transmission-daemon/settings.json"

func execRemoteTorrentCmd(cmd string) (response string) {
	remoteResult := executeRemoteCmd(cmd, structures.VPNPIRemoteConnectConfig)
	if remoteResult.Stdout == "" && remoteResult.Stderr != "" {
		response = remoteResult.Stderr
	} else {
		response = remoteResult.Stdout
	}

	return response
}

func getTorrents() (response string) {
	cmd := "docker exec vpnission transmission-remote --list"

	return execRemoteTorrentCmd(cmd)
}

func addTorrents(torrentLink string, paused bool) (response string) {
	// slack 'markdown's URLs with '<link|text>' so clip these off
	if strings.HasPrefix(torrentLink, "<http://magnet/") {
		torrentLink = "magnet:" + strings.TrimLeft(torrentLink, "<http://magnet/")
	} else {
		torrentLink = strings.TrimLeft(torrentLink, "<")
	}

	if indexPipe := strings.Index(torrentLink, "|"); indexPipe > 0 {
		torrentLink = torrentLink[:indexPipe]
	} else {
		torrentLink = strings.TrimRight(torrentLink, ">")
	}
	response = fmt.Sprintf(":star2: adding %s\n", torrentLink)

	// Add a torrent
	pausedParam := "--no-start-paused"
	if paused {
		pausedParam = "--start-paused"
	}
	cmd := fmt.Sprintf("docker exec vpnission transmission-remote %s -a \"%s\"",
		pausedParam, torrentLink)

	return response + execRemoteTorrentCmd(cmd) + getTorrents()
}

func deleteTorrents(torrentIDStr string) (result string) {
	result = fmt.Sprintf(":x: deleting torrent #%s\n", torrentIDStr)

	cmd := fmt.Sprintf("docker exec vpnission transmission-remote -t %s -r",
		torrentIDStr)

	remoteResult := executeRemoteCmd(cmd, structures.VPNPIRemoteConnectConfig)
	if remoteResult.Stdout == "" && remoteResult.Stderr != "" {
		result += remoteResult.Stderr
	} else {
		result += remoteResult.Stdout
	}

	return execRemoteTorrentCmd(cmd) + getTorrents()
}

func torrentCommand(cmd []string) (result string) {
	result = ":closed_lock_with_key: unable to talk to raspberrypi..."

	if cmd[0] == "trans" {
		result = getTorrents()
	} else if cmd[0] == "tranc" {
		if len(cmd) == 1 {
			result = "Usage: `tranc <Torrent link>`"
		} else {
			paused := false
			result = addTorrents(cmd[1], paused)
		}
	} else if cmd[0] == "tranp" {
		if len(cmd) == 1 {
			result = "Usage: `tranp <Torrent link>`"
		} else {
			paused := true
			result = addTorrents(cmd[1], paused)
		}
	} else if cmd[0] == "trand" {
		if len(cmd) == 1 {
			result = "Usage: `trand <Torrent ID>`"
		} else {
			result = deleteTorrents(cmd[1])
		}
	}

	result = strings.Replace(result,
		"localhost:9091/transmission/rpc/",
		"http://vpnpi.fritz.box:9091/transmission/web/", 1)
	return result
}

func transmissionSettingsAreSane(internalIP string) bool {
	result := false

	// match for correct ipv4 IP bind & broken ipv6 bind
	cmd := `docker exec vpnission grep -e '"bind-address-ipv4": "` +
		internalIP + `",' ` + `-e '"bind-address-ipv6": "fe80::",' ` +
		transmissionSettingsPath
	remoteResult := executeRemoteCmd(cmd, structures.VPNPIRemoteConnectConfig)
	Logger.Printf("transmission settings: %s", remoteResult.Stdout)
	if remoteResult.Err == nil {
		if len(strings.Split(remoteResult.Stdout, "\n")) == 3 { // incl trailing \n
			result = true
		} else if internalIP != "" {
			Logger.Printf("FIXing Transmission settings: %s not found...", internalIP)

			response := updateVpnPiTunnel("NL_28") // other wireguard endpoint is NL_88

			if strings.HasPrefix(response, "Updated") {
				result = true
			}
		}
	} else {
		Logger.Printf("ERR: transmission settings not sane -> %s", remoteResult.Err.Error())
	}

	return result
}
