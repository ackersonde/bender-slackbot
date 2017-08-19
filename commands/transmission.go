package commands

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/odwrtw/transmission"
)

func getTorrents(t *transmission.Client) (result string) {
	// Get all torrents
	torrents, err := t.GetTorrents()
	if err == nil {
		result = ":transmission: <http://" + raspberryPIIP + ":9091/transmission/web/|Running RaspberryPI Torrent(s)>\n"
		for _, listTorrent := range torrents {
			status := ":arrows_counterclockwise:"
			switch listTorrent.Status {
			case transmission.StatusStopped:
				status = ":black_square_for_stop:"
			case transmission.StatusDownloading:
				status = ":arrow_double_down:"
			case transmission.StatusSeeding:
				status = ":cinema:"
			}

			percentComplete := strconv.FormatFloat(listTorrent.PercentDone*100, 'f', 0, 64)
			result += status + " *" + strconv.Itoa(listTorrent.ID) + "*: " +
				listTorrent.Name + " *" + percentComplete + "%* [S: " +
				strconv.Itoa(listTorrent.PeersSendingToUs) + "]\n"
		}
	} else {
		fmt.Printf("\nGetTorrents err: %v", err)
	}

	return result
}

func addTorrents(t *transmission.Client, torrentLink string) (result string) {
	// slack 'markdown's URLs with '<link|text>' so clip these off
	if strings.HasPrefix(torrentLink, "<") {
		torrentLink = strings.TrimLeft(torrentLink, "<")
	}
	if strings.HasSuffix(torrentLink, ">") {
		torrentLink = strings.TrimRight(torrentLink, ">")
	}
	if indexPipe := strings.Index(torrentLink, "|"); indexPipe > 0 {
		torrentLink = torrentLink[:indexPipe]
	}

	torrentLink = strings.Replace(torrentLink, "http://magnet/", "magnet:", -1)
	result = fmt.Sprintf(":star2: adding %s\n", torrentLink)

	// Add a torrent
	_, err := t.Add(torrentLink)
	if err != nil {
		fmt.Printf("\nAdd err: %v", err)
	}

	result += getTorrents(t)

	return result
}

func deleteTorrents(t *transmission.Client, torrentIDStr string) (result string) {
	torrentID, err := strconv.Atoi(torrentIDStr)
	if err != nil {
		fmt.Printf("\nRemove err: %v", err)
		return fmt.Sprintf("Unable to remove torrent ID #%s. Is it a valid ID?", torrentIDStr)
	}

	result = fmt.Sprintf(":x: deleting torrent #%d\n", torrentID)
	torrentToDelete := &transmission.Torrent{ID: torrentID}
	removeErr := t.RemoveTorrents([]*transmission.Torrent{torrentToDelete}, false)
	if err != nil {
		fmt.Printf("\nRemove err: %v", removeErr)
	}

	result += getTorrents(t)
	return result
}

func torrentCommand(cmd []string) (result string) {
	defer func() { //catch or finally
		if err := recover(); err != nil { //catch
			fmt.Fprintf(os.Stderr, "Exception: %v\n", err)
			result = "\nlost connection to Transmission Daemon!\n" + vpnTunnelCmds("status")
		}
	}()

	result = ":closed_lock_with_key: No tunnel exists! Try `vpnc` first..."

	if runningFritzboxTunnel() {
		if raspberryPIIP == "" {
			raspberryPIIP = "raspberrypi.fritz.box"
		}

		// Connect to Transmission RPC daemon
		conf := transmission.Config{
			Address: "http://" + raspberryPIIP + ":9091/transmission/rpc",
		}
		t, err := transmission.New(conf)
		if err != nil {
			fmt.Printf("\nNew err: %v", err)
		}

		if cmd[0] == "trans" {
			result = getTorrents(t)
		} else if cmd[0] == "tranc" {

			if len(cmd) == 1 {
				result = "Usage: `tranc <Torrent link>`"
			} else {
				result = addTorrents(t, cmd[1])
			}
		} else if cmd[0] == "trand" {
			if len(cmd) == 1 {
				result = "Usage: `trand <Torrent ID>`"
			} else {
				result = deleteTorrents(t, cmd[1])
			}
		}

		tunnelIdleSince = time.Now()
	}

	return result
}
