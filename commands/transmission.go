package commands

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/odwrtw/transmission"
)

func getTorrents(t *transmission.Client) (result string) {
	// Get all torrents
	torrents, err := t.GetTorrents()
	if err == nil {
		result = ":transmission: <http://" +
			vpnPIRemoteConnectConfig.HostName +
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
	conf := transmission.Config{
		Address: "http://" + vpnPIRemoteConnectConfig.HostName + ":9091/transmission/rpc",
	}
	t, err := transmission.New(conf)
	if err != nil {
		Logger.Printf("\nNew err: %v", err)
	}

	if cmd[0] == "trans" {
		result = getTorrents(t)
	} else if cmd[0] == "tranc" {
		if len(cmd) == 1 {
			result = "Usage: `tranc <Torrent link>`"
		} else {
			ensureTransmissionBind()
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

func ensureTransmissionBind() string {
	response := "Unable to update :transmission: ipv4 bind"
	transmissionSettingsPath := "/home/ubuntu/.config/transmission-daemon/settings.json"

	cmd := "VPN_IP=`ip address | grep '10\\.' | awk '{print $2}' | cut -f1 -d/`; " +
		`grep "\"bind-address-ipv4\": \"$VPN_IP\"" ` + transmissionSettingsPath +
		" || echo $VPN_IP"
	Logger.Printf("VPN_IP running %s", cmd)
	remoteResult := executeRemoteCmd(cmd, vpnPIRemoteConnectConfig)
	// ^-- returns e.g. "bind-address-ipv4": "10.1.8.75", if found
	// else 10.1.8.75 if *not* found

	internalIP := strings.TrimSuffix(remoteResult.stdout, "\n")
	// TODO: while?
	if internalIP == "" {
		time.Sleep(10 * time.Second)
		remoteResult = executeRemoteCmd(cmd, vpnPIRemoteConnectConfig)
		internalIP = strings.TrimSuffix(remoteResult.stdout, "\n")
	}

	if remoteResult.err == nil && internalIP != "" &&
		!strings.Contains(internalIP, "bind-address-ipv4") {
		Logger.Printf("internal VPN IP: %s", internalIP)

		sedCmd := `sed -rie 's/"bind-address-ipv4": "(.*)"/"bind-address-ipv4": "` +
			internalIP + `"/' `
		cmd = `sudo service transmission-daemon stop && ` +
			sedCmd + transmissionSettingsPath +
			` && sudo service transmission-daemon start`

		Logger.Printf("exec VPN PI update: %s", cmd)
		remoteResult = executeRemoteCmd(cmd, vpnPIRemoteConnectConfig)
		if remoteResult.err == nil {
			response = "Changed :transmission: ipv4 bind: " + internalIP
		}
	} else if remoteResult.err != nil {
		response += "\n with " + remoteResult.err.Error()
	} else if strings.Contains(internalIP, "bind-address-ipv4") {
		response = ":transmission: ipv4 bind already correctly set: " + internalIP
	}

	return response
}
