package commands

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/odwrtw/transmission"
)

func getTorrents(t *transmission.Client) (result string) {
	// Get all torrents
	torrents, err := t.GetTorrents()
	if err == nil {
		for _, listTorrent := range torrents {
			result += strconv.Itoa(listTorrent.ID) + ": " + listTorrent.Name + "\n"
		}
	} else {
		fmt.Printf("\nGetTorrents err: %v", err)
	}

	return result
}

func addTorrents(t *transmission.Client, params []string) (result string) {
	result = fmt.Sprintf("going to add your torrent %v", params)

	// Add a torrent
	//torrent, err := t.Add("http://torrent.ubuntu.com:6969/file?info_hash=%BFo%2B%E5I%A8%AC%A5wf8%B5%9B%2B%CAS%D7%BB%C7H")
	_, err := t.Add(params[0])
	if err != nil {
		fmt.Printf("\nAdd err: %v", err)
	}

	result = getTorrents(t)
	return result
}

func deleteTorrents(t *transmission.Client, params []string) (result string) {
	result = fmt.Sprintf("going to delete your torrent %v", params)

	// TODO find it (by ID?)
	torrentID, _ := strconv.Atoi(params[0])
	torrentToDelete := &transmission.Torrent{ID: torrentID}

	// Remove it
	err := t.RemoveTorrents([]*transmission.Torrent{torrentToDelete}, true)
	if err != nil {
		fmt.Printf("\nRemove err: %v", err)
	}

	// Get session informations
	t.Session.Update()

	result = getTorrents(t)
	return result
}

func torrentCommand(cmd ...string) (result string) {
	defer func() { //catch or finally
		if err := recover(); err != nil { //catch
			fmt.Fprintf(os.Stderr, "Exception: %v\n", err)
			result = "\nlost connection to Transmission Daemon!\n" + vpnTunnelCmds("status")
		}
	}()

	result = "No VPN Tunnel established! Try `vpnc` first..."

	tunnelStatus := vpnTunnelCmds("status")
	if strings.Contains(tunnelStatus, "inet 192.168.178.201/32 scope global tun0") {
		// Connect to Transmission RPC daemon
		conf := transmission.Config{
			Address: "http://192.168.178.38:9091/transmission/rpc",
		}
		t, err := transmission.New(conf)
		if err != nil {
			fmt.Printf("\nNew err: %v", err)
		}

		if cmd[0] == "trans" {
			result = "RaspberryPI Transmission Torrent(s):\n"
			result += getTorrents(t)
		} else if cmd[0] == "tranc" {
			result += addTorrents(t, cmd[1:])
		} else if cmd[0] == "trand" {
			result += deleteTorrents(t, cmd[1:])
		}
	}

	return result
}
