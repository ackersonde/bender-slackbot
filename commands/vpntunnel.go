package commands

import (
	"encoding/json"
	"fmt"
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

	results := make(chan string, 10)
	timeout := time.After(10 * time.Second)
	go func() {
		// get both ipv4+ipv6 internet addresses
		cmd := "curl https://ipleak.net/json/"
		details := RemoteCmd{Host: raspberryPIIP, Cmd: cmd}

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
					cmd := "dig " + vpnGateway + " A +short"
					details := RemoteCmd{Host: raspberryPIIP, Cmd: cmd}

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
					fmt.Println("Timed out on dig " + vpnGateway + "!")
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
			details := RemoteCmd{Host: raspberryPIIP, Cmd: cmd}

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
		details := RemoteCmd{Host: raspberryPIIP, Cmd: cmd}

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

	return response
}
