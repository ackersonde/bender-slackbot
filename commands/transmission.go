package commands

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ackersonde/bender-slackbot/structures"
	"github.com/odwrtw/transmission"
)

var transmissionSettingsPath = "/root/.config/transmission-daemon/settings.json"

func getTorrents(t *transmission.Client) (result string) {
	// Get all torrents
	torrents, err := t.GetTorrents()
	if err == nil {
		result = ":transmission: <http://" +
			structures.VPNPIRemoteConnectConfig.HostName +
			":9091/transmission/web/|Running RaspberryPI Torrent(s)>\n"

		for _, listTorrent := range torrents {
			status := ":arrows_counterclockwise:"
			info := "[S: " + strconv.Itoa(listTorrent.PeersSendingToUs) + "]\n"

			switch listTorrent.Status {
			case transmission.StatusStopped:
				status = ":black_square_for_stop:"
			case transmission.StatusDownloading:
				status = ":arrow_double_down:"
			case transmission.StatusSeeding:
				status = ":cinema:"
				seedRatio := fmt.Sprintf("%.1f", listTorrent.UploadRatio)
				info = "[L: " + strconv.Itoa(listTorrent.PeersGettingFromUs) +
					"] (Ratio: " + seedRatio + ")\n"
			}

			percentComplete := strconv.FormatFloat(listTorrent.PercentDone*100, 'f', 0, 64)
			result += status + " *" + strconv.Itoa(listTorrent.ID) + "*: " +
				listTorrent.Name + " *" + percentComplete + "%* " + info
		}
	} else {
		Logger.Printf("\nGetTorrents err: %v", err)
	}

	return result
}

func addTorrents(t *transmission.Client, torrentLink string, paused bool) string {
	// slack 'markdown's URLs with '<link|text>' so clip these off
	if strings.HasPrefix(torrentLink, "<") {
		torrentLink = strings.TrimLeft(torrentLink, "<http://")
	}
	if strings.HasSuffix(torrentLink, ">") {
		torrentLink = strings.TrimRight(torrentLink, ">")
	}
	if indexPipe := strings.Index(torrentLink, "|"); indexPipe > 0 {
		torrentLink = torrentLink[:indexPipe]
	}

	torrentLink = strings.Replace(torrentLink, "magnet/", "magnet:", -1)
	result := fmt.Sprintf(":star2: adding %s\n", torrentLink)

	// Add a torrent

	args := transmission.AddTorrentArg{Filename: torrentLink, Paused: paused}
	_, err := t.AddTorrent(args)
	if err != nil {
		Logger.Printf("\nAdd err: %v", err)
	}

	result += getTorrents(t)

	return result
}

func deleteTorrents(t *transmission.Client, torrentIDStr string) (result string) {
	torrentID, err := strconv.Atoi(torrentIDStr)
	if err != nil {
		Logger.Printf("\nRemove err: %v", err)
		return fmt.Sprintf("Unable to remove torrent ID #%s. Is it a valid ID?", torrentIDStr)
	}

	result = fmt.Sprintf(":x: deleting torrent #%d\n", torrentID)
	torrentToDelete := &transmission.Torrent{ID: torrentID}
	removeErr := t.RemoveTorrents([]*transmission.Torrent{torrentToDelete}, false)
	if removeErr != nil {
		Logger.Printf("\nRemove err: %v", removeErr)
	}

	result += getTorrents(t)
	return result
}

func torrentCommand(cmd []string) (result string) {
	result = ":closed_lock_with_key: unable to talk to raspberrypi..."

	// Connect to Transmission RPC daemon
	endpoint := structures.VPNPIRemoteConnectConfig.HostEndpoints[0]
	conf := transmission.Config{
		Address: "http://" + strings.TrimSuffix(endpoint, ":22") + ":9091/transmission/rpc",
	}
	t, err := transmission.New(conf)
	if err != nil {
		Logger.Printf("\nUnable to instantiate Transmission RPC client: %v", err)
	}

	if cmd[0] == "trans" {
		result = getTorrents(t)
	} else if cmd[0] == "tranc" {
		if len(cmd) == 1 {
			result = "Usage: `tranc <Torrent link>`"
		} else {
			paused := false
			result = addTorrents(t, cmd[1], paused)
		}
	} else if cmd[0] == "tranp" {
		if len(cmd) == 1 {
			result = "Usage: `tranp <Torrent link>`"
		} else {
			paused := true
			result = addTorrents(t, cmd[1], paused)
		}
	} else if cmd[0] == "trand" {
		if len(cmd) == 1 {
			result = "Usage: `trand <Torrent ID>`"
		} else {
			result = deleteTorrents(t, cmd[1])
		}
	}

	tunnelIdleSince = time.Now()

	return result
}

func transmissionSettingsAreSane(internalIP string) bool {
	result := false

	// match for correct ipv4 IP bind & broken ipv6 bind
	cmd := `sudo docker exec vpnission grep -e '"bind-address-ipv4": "` +
		internalIP + `",' ` + `-e '"bind-address-ipv6": "fe80::",' ` +
		transmissionSettingsPath
	remoteResult := executeRemoteCmd(cmd, structures.VPNPIRemoteConnectConfig)
	Logger.Printf("transmission settings: %s", remoteResult.Stdout)
	if remoteResult.Err == nil {
		if len(strings.Split(remoteResult.Stdout, "\n")) == 3 { // incl trailing \n
			result = true
		} else if internalIP != "" {
			Logger.Printf("FIXing Transmission settings: %s not found...", internalIP)

			bestVPNServer := findBestVPNServer(VPNCountry)
			response := updateVpnPiTunnel(bestVPNServer.Domain)

			if strings.HasPrefix(response, "Updated") {
				result = true
			}
		}
	} else {
		Logger.Printf("ERR: transmission settings not sane -> %s", remoteResult.Err.Error())
	}

	return result
}
