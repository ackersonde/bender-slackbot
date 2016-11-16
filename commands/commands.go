package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/nlopes/slack"
	"github.com/odwrtw/transmission"
)

func curlTransmission(command ...string) (result string) {
	defer func() { //catch or finally
		if err := recover(); err != nil { //catch
			fmt.Fprintf(os.Stderr, "Exception: %v\n", err)
			result = "\nlost connection to Transmission Daemon!\n" + vpnTunnelCmds("status")
		}
	}()

	// Simple client
	conf := transmission.Config{
		Address: "http://192.168.178.38:9091/transmission/rpc",
	}
	t, err := transmission.New(conf)
	if err != nil {
		fmt.Printf("\nNew err: %v", err)
	}

	// Get all torrents
	torrents, err := t.GetTorrents()
	if err != nil {
		fmt.Printf("\nGet err: %v", err)
	}
	fmt.Printf("\nlist of torrents: %v", torrents)

	// Add a torrent
	torrent, err := t.Add("http://torrent.ubuntu.com:6969/file?info_hash=%BFo%2B%E5I%A8%AC%A5wf8%B5%9B%2B%CAS%D7%BB%C7H")
	if err != nil {
		fmt.Printf("\nAdd err: %v", err)
	}

	// Update is information
	torrent.Update()
	fmt.Printf("\nUpdate NewTorrent: %v", torrent)

	// Remove it
	err = t.RemoveTorrents([]*transmission.Torrent{torrent}, true)
	if err != nil {
		fmt.Printf("\nRemove err: %v", err)
	}

	// Get session informations
	t.Session.Update()

	torrents2, err := t.GetTorrents()
	if err == nil {
		for _, listTorrent := range torrents2 {
			result += strconv.Itoa(listTorrent.ID) + ": " + listTorrent.Name + "\n"
		}
	} else {
		fmt.Printf("\nGetFinal err: %v", err)
	}

	//t.Session.Close()
	return result
}

func vpnTunnelCmds(command ...string) string {
	if command[0] != "status" {
		cmd := exec.Command(command[0])

		args := len(command)
		if args > 1 {
			cmd = exec.Command(command[0], command[1])
		}

		errStart := cmd.Start()
		if errStart != nil {
			os.Stderr.WriteString(errStart.Error())
		}

		if errWait := cmd.Wait(); errWait != nil {
			fmt.Println(errWait)
		}
	}

	/* Here's the next cmd to get setup
			# ip a show tun0
			9: tun0: <POINTOPOINT,MULTICAST,NOARP,UP,LOWER_UP> mtu 1024 qdisc pfifo_fast state UNKNOWN group default qlen 500 link/none
	    		inet 192.168.178.201/32 scope global tun0
	       	valid_lft forever preferred_lft forever
			# vpnc-disconnect
				Terminating vpnc daemon (pid: 174)
			# ip a show tun0
				Device "tun0" does not exist.
	*/
	tun0StatusCmd := "/sbin/ip a show tun0 | /bin/grep tun0 | /usr/bin/tail -1"
	tunnel, err := exec.Command("/bin/bash", "-c", tun0StatusCmd).Output()
	if err != nil {
		fmt.Printf("Failed to execute command: %s", tun0StatusCmd)
	}

	tunnelStatus := string(tunnel)
	if len(tunnelStatus) == 0 {
		tunnelStatus = "Tunnel offline."
	}

	return tunnelStatus
}

// CheckCommand is now commented
func CheckCommand(api *slack.Client, rtm *slack.RTM, slackMessage slack.Msg, command string) {
	if command == "do" {
		ListDODroplets(rtm)
	} else if command == "sw" {
		response := ":partly_sunny_rain: <https://www.wunderground.com/cgi-bin/findweather/getForecast?query=48.3,11.35#forecast-graph|10-day forecast Schwabhausen>"
		params := slack.PostMessageParameters{AsUser: true}
		api.PostMessage(slackMessage.Channel, response, params)
	} else if command == "vpnc" {
		result := vpnTunnelCmds("/usr/sbin/vpnc-connect", "fritzbox")
		rtm.SendMessage(rtm.NewOutgoingMessage(result, slackMessage.Channel))
	} else if command == "vpnd" {
		result := vpnTunnelCmds("/usr/sbin/vpnc-disconnect")
		rtm.SendMessage(rtm.NewOutgoingMessage(result, slackMessage.Channel))
	} else if command == "vpns" {
		result := vpnTunnelCmds("status")
		rtm.SendMessage(rtm.NewOutgoingMessage(result, slackMessage.Channel))
	} else if command == "trans" {
		result := torrentCommand("trans")
		rtm.SendMessage(rtm.NewOutgoingMessage(result, slackMessage.Channel))
	} else if command == "trand" {
		// TODO delete indicated torrent
		result := "Deleting " + command
		rtm.SendMessage(rtm.NewOutgoingMessage(result, slackMessage.Channel))
	} else if command == "trang" {
		// TODO grab indicated magnet link
		result := "Fetching magnet " + command
		rtm.SendMessage(rtm.NewOutgoingMessage(result, slackMessage.Channel))
	} else if command == "help" {
		response := "`sw`: Schwabhausen weather\n" +
			"`vpn[c|s|d]`: [C]onnect, [S]tatus, [D]rop VPN tunnel to fritz.box\n" +
			"`tran[c|s|d]`: [C]reate, [S]tatus, [D]rop torrents on RaspberryPI Transmission\n"
		params := slack.PostMessageParameters{AsUser: true}
		api.PostMessage(slackMessage.Channel, response, params)
	} else {
		callingUserProfile, _ := api.GetUserInfo(slackMessage.User)
		rtm.SendMessage(rtm.NewOutgoingMessage("whaddya say <@"+callingUserProfile.Name+">? "+command+"?", slackMessage.Channel))
	}
}
