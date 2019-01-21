package commands

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/nlopes/slack"
)

var tunnelOnTime time.Time
var tunnelIdleSince time.Time
var maxTunnelIdleTime = float64(5 * 60) // 5 mins in seconds
var piHostKey = "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBKqLtosnMy7YnC+FXAxqevMgOGPkz0tPHYcfZlA+sfWLW49wCbzdYon3F47QjqzYA8Bx8J/FAdU6VB3UHKfmgYg="

// RaspberryPIPrivateTunnelChecks ensures PrivateTunnel vpn connection
// on PI is up and working properly
func RaspberryPIPrivateTunnelChecks(userCall bool) string {
	tunnelUp := ""
	response := ":openvpn: PI status: DOWN :rotating_light:"

	if runningFritzboxTunnel() {
		results := make(chan string, 10)
		timeout := time.After(10 * time.Second)
		go func() {
			// get both ipv4+ipv6 internet addresses
			cmd := "curl https://ipleak.net/json/"
			details := RemoteCmd{Host: raspberryPIIP, HostKey: piHostKey, Username: os.Getenv("piUser"), Password: os.Getenv("piPass"), Cmd: cmd}

			remoteResult := executeRemoteCmd(details)

			tunnelIdleSince = time.Now()
			results <- remoteResult.stdout
		}()

		type IPInfoResponse struct {
			IP          string
			CountryCode string `json:"country_code"`
		}
		var jsonRes IPInfoResponse

		select {
		case res := <-results:
			if res != "" {
				err := json.Unmarshal([]byte(res), &jsonRes)
				if err != nil {
					fmt.Printf("unable to parse JSON string (%v)\n%s\n", err, res)
				} else {
					fmt.Printf("ipleak.net: %v\n", jsonRes)
				}
				if jsonRes.CountryCode == "NL" || jsonRes.CountryCode == "SE" {
					resultsDig := make(chan string, 10)
					timeoutDig := time.After(10 * time.Second)
					// ensure home.ackerson.de is DIFFERENT than PI IP address!
					go func() {
						cmd := "dig home.ackerson.de A +short"
						details := RemoteCmd{Host: raspberryPIIP, HostKey: piHostKey, Username: os.Getenv("piUser"), Password: os.Getenv("piPass"), Cmd: cmd}

						remoteResult := executeRemoteCmd(details)

						tunnelIdleSince = time.Now()
						resultsDig <- remoteResult.stdout
					}()
					select {
					case resComp := <-resultsDig:
						fmt.Println("dig results: " + resComp)
						lines := strings.Split(resComp, "\n")
						if lines[1] != jsonRes.IP {
							tunnelUp = jsonRes.IP
						}
					case <-timeoutDig:
						fmt.Println("Timed out on dig home.ackerson.de!")
					}
				}
			}
		case <-timeout:
			fmt.Println("Timed out on curl ipleak.net!")
		}

		// Tunnel should be OK. Now double check iptables to ensure that
		// ALL Internet requests are running over OpenVPN!
		if tunnelUp != "" {
			resultsIPTables := make(chan string, 10)
			timeoutIPTables := time.After(5 * time.Second)
			go func() {
				cmd := "sudo iptables -L OUTPUT -v --line-numbers | grep all"
				details := RemoteCmd{Host: raspberryPIIP, HostKey: piHostKey, Username: os.Getenv("piUser"), Password: os.Getenv("piPass"), Cmd: cmd}

				remoteResult := executeRemoteCmd(details)

				tunnelIdleSince = time.Now()
				resultsIPTables <- remoteResult.stdout
			}()
			select {
			case resIPTables := <-resultsIPTables:
				lines := strings.Split(resIPTables, "\n")

				for idx, oneLine := range lines {
					switch idx {
					case 0:
						if !strings.Contains(oneLine, "ACCEPT     all  --  any    tun0    anywhere") {
							tunnelUp = ""
						}
					case 1:
						if !strings.Contains(oneLine, "ACCEPT     all  --  any    eth0    anywhere             192.168.178.0/24") {
							tunnelUp = ""
						}
					case 2:
						if !strings.Contains(oneLine, "DROP       all  --  any    eth0    anywhere             anywhere") {
							tunnelUp = ""
						}
					}
				}
			case <-timeoutIPTables:
				fmt.Println("Timed out on `iptables -L OUTPUT`!")
			}
		} else {
			cmd := "sudo service openvpn@AMD restart && sudo service transmission-daemon restart"
			details := RemoteCmd{Host: raspberryPIIP, HostKey: piHostKey, Username: os.Getenv("piUser"), Password: os.Getenv("piPass"), Cmd: cmd}

			remoteResult := executeRemoteCmd(details)
			fmt.Println("restarting VPN & Transmission: " + remoteResult.stdout)
		}

		if tunnelUp != "" {
			response = ":openvpn: PI status: UP :raspberry_pi: @ " + tunnelUp
		}

		if !userCall {
			customEvent := slack.RTMEvent{Type: "RaspberryPIPrivateTunnelChecks", Data: response}
			rtm.IncomingEvents <- customEvent
		}
	} else {
		response = "Unable to connect to Fritz!Box tunnel to check :openvpn:"
	}
	return response
}

// CheckPiDiskSpace now exported
func CheckPiDiskSpace(path string) string {
	userCall := true
	if path == "---" {
		path = ""
		userCall = false
	} else {
		path = strings.TrimPrefix(path, "/")
	}

	diskUsage := "du -ah \"" + piSDCardPath + path
	diskUsageAll := diskUsage + "*\""
	diskUsageOne := diskUsage + "\""
	cmd := "[ \"$(ls -A '" + piSDCardPath + "')\" ] && " + diskUsageAll + " || " + diskUsageOne
	fmt.Println("chk disk usage: " + cmd)
	details := RemoteCmd{Host: raspberryPIIP, HostKey: piHostKey, Username: os.Getenv("piUser"), Password: os.Getenv("piPass"), Cmd: cmd}

	remoteResult := executeRemoteCmd(details)
	response := remoteResult.stdout
	tunnelIdleSince = time.Now()
	if remoteResult.stdout == "" && remoteResult.stderr != "" && strings.HasPrefix(remoteResult.stderr, "du: cannot access ‘/home/pi/torrents/*’: No such file or directory") {
		response = remoteResult.stderr
	} else {
		response = strings.Replace(response, piSDCardPath+path, "", -1)
		response = ":raspberry_pi: *SD Card Disk Usage* @ `" + piSDCardPath + path + "`\n" + response
	}
	cmd = "df -h /root/"
	details = RemoteCmd{Host: raspberryPIIP, HostKey: piHostKey, Username: os.Getenv("piUser"), Password: os.Getenv("piPass"), Cmd: cmd}
	remoteResultDF := executeRemoteCmd(details)
	response += "\n\n" + remoteResultDF.stdout

	if !userCall {
		customEvent := slack.RTMEvent{Type: "CheckPiDiskSpace", Data: response}
		rtm.IncomingEvents <- customEvent
	} else {
		tunnelIdleSince = time.Now()
	}

	return response
}

// DeleteTorrentFile now exported
func DeleteTorrentFile(filename string) string {
	var response string

	if filename == "*" || filename == "" || strings.Contains(filename, "../") {
		response = "Please enter an existing filename - try `fsck`"
	} else {
		path := piSDCardPath + filename

		var deleteCmd string
		cmd := "test -d \"" + path + "\" && echo 'Yes'"
		details := RemoteCmd{Host: raspberryPIIP, HostKey: piHostKey, Username: os.Getenv("piUser"), Password: os.Getenv("piPass"), Cmd: cmd}

		remoteResult := executeRemoteCmd(details)
		tunnelIdleSince = time.Now()
		if strings.HasPrefix(remoteResult.stdout, "Yes") {
			deleteCmd = "rm -Rf \"" + path + "\""
		} else {
			deleteCmd = "rm \"" + path + "\""
		}

		details = RemoteCmd{Host: raspberryPIIP, HostKey: piHostKey, Username: os.Getenv("piUser"), Password: os.Getenv("piPass"), Cmd: deleteCmd}

		remoteResultDelete := executeRemoteCmd(details)
		tunnelIdleSince = time.Now()
		if remoteResultDelete.stderr != "" {
			response = remoteResultDelete.stderr
		} else {
			response = remoteResultDelete.stdout
		}
	}

	return response
}

// MoveTorrentFile now exported
func MoveTorrentFile(api *slack.Client, filename string) {
	if filename == "" || strings.Contains(filename, "../") || strings.HasPrefix(filename, "/") {
		rtm.IncomingEvents <- slack.RTMEvent{Type: "MoveTorrent", Data: "Please enter an existing filename - try `fsck`"}
	} else {
		// detox filenames => http://detox.sourceforge.net/ | https://linux.die.net/man/1/detox
		renameCmd := "cd " + piSDCardPath + "; rm *.log; detox -r *"
		renameDetails := RemoteCmd{Host: raspberryPIIP, HostKey: piHostKey, Username: os.Getenv("piUser"), Password: os.Getenv("piPass"), Cmd: renameCmd}
		executeRemoteCmd(renameDetails)

		moveCmd := "cd " + piSDCardPath + "; find . -type f -exec curl -g --ftp-create-dirs -u ftpuser:abc123 -T \"{}\" \"ftp://192.168.178.1/backup/DLNA/torrents/{}\" \\; > ftp.log 2>&1"
		log.Println(moveCmd)
		go func() {
			details := RemoteCmd{Host: raspberryPIIP, HostKey: piHostKey, Username: os.Getenv("piUser"), Password: os.Getenv("piPass"), Cmd: moveCmd}
			var result string
			remoteResult := executeRemoteCmd(details)
			tunnelIdleSince = time.Now()
			log.Printf("%v:%v", details, remoteResult)

			result = "Successfully moved `" + filename + "` to `" + piUSBMountPath + "`"
			rtm.IncomingEvents <- slack.RTMEvent{Type: "MoveTorrent", Data: result}

			cleanPITorrentsCmd := "cd " + piSDCardPath + "; rm -Rf *;"
			details = RemoteCmd{Host: raspberryPIIP, HostKey: piHostKey,
				Username: os.Getenv("piUser"),
				Password: os.Getenv("piPass"),
				Cmd:      cleanPITorrentsCmd}
			remoteResult = executeRemoteCmd(details)
		}()

		params := slack.MsgOptionAsUser(true)
		api.PostMessage(SlackReportChannel, slack.MsgOptionText("running `"+moveCmd+"`", true), params)
		//reportMoveProgress(api)
	}
}

func reportMoveProgress(api *slack.Client) {
	historyParams := new(slack.HistoryParameters)
	historyParams.Latest = ""
	historyParams.Count = 1
	historyParams.Inclusive = true
	lastMsgID := ""
	msgHistory, _ := api.GetChannelHistory(SlackReportChannel, *historyParams)
	for _, msg := range msgHistory.Messages {
		lastMsgID = msg.Timestamp
	}

	remoteResults := make(chan RemoteResult, 1)
	timeout := time.After(10 * time.Second)
	notDone := true

	for notDone {
		go func() {
			progressCmd := "progress"
			progressDetails := RemoteCmd{Host: raspberryPIIP, HostKey: piHostKey, Username: "pi", Password: "king8tut", Cmd: progressCmd}

			remoteResults <- executeRemoteCmd(progressDetails)
		}()

		// reset tunnel idle time as user may want to see progress of move
		tunnelIdleSince = time.Now()

		select {
		case res := <-remoteResults:
			// update msg with progress: https://api.slack.com/methods/chat.update
			// so there aren't 385 msgs with 2% 2% 3% ...
			api.UpdateMessage(SlackReportChannel, lastMsgID, slack.MsgOptionText(res.stdout, true))
			if strings.Contains(res.stderr, "No command currently running") {
				notDone = false
			} else {
				time.Sleep(time.Second * 5)
			}
		case <-timeout:
			fmt.Println("Timed out!")
		}
	}
}

// DisconnectIdleTunnel is now commented
func DisconnectIdleTunnel() {
	msg := ":closed_lock_with_key: UP since: " + tunnelOnTime.Format("Mon, Jan 2 15:04") + " IDLE for "

	if !tunnelOnTime.IsZero() {
		currentIdleTime := time.Now().Sub(tunnelIdleSince)
		stringCurrentIdleTimeSecs := strconv.FormatFloat(currentIdleTime.Seconds(), 'f', 0, 64)
		if currentIdleTime.Seconds() > maxTunnelIdleTime {
			vpnTunnelCmds("/usr/sbin/vpnc-disconnect")
			msg += stringCurrentIdleTimeSecs + "secs => disconnected!"
			rtm.SendMessage(rtm.NewOutgoingMessage(msg, SlackReportChannel))
		}
	}
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
		} else if errWait := cmd.Wait(); errWait != nil {
			os.Stderr.WriteString(errWait.Error())
		}

		if strings.HasSuffix(command[0], "vpnc-connect") {
			now := time.Now()
			tunnelOnTime, tunnelIdleSince = now, now
		} else if strings.HasSuffix(command[0], "vpnc-disconnect") {
			emptyTime := *new(time.Time)
			tunnelOnTime, tunnelIdleSince = emptyTime, emptyTime
		}
	}

	/* Here's the next cmd to get setup
			# ip a show tun0
			9: tun0: <POINTOPOINT,MULTICAST,NOARP,UP,LOWER_UP> mtu 1024 qdisc
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

	return ":closed_lock_with_key: " + tunnelStatus + " IDLE since: " + tunnelIdleSince.Format("Mon Jan _2 15:04")
}

func runningFritzboxTunnel() bool {
	up := isFritzboxTunnelUp()
	//up := true
	if !up { // attempt to establish connection
		vpnTunnelCmds("/usr/sbin/vpnc-connect", "fritzbox")
		if up = isFritzboxTunnelUp(); !up {
			rtm.SendMessage(rtm.NewOutgoingMessage(
				":closed_lock_with_key: Unable to tunnel to Fritz!Box", ""))
		}
	}

	return up
}

func isFritzboxTunnelUp() bool {
	status := false

	tunnelStatus := vpnTunnelCmds("status")
	if strings.Contains(tunnelStatus, "192.168.178.201") {
		status = true
	}

	return status
}
