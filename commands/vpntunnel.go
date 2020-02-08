package commands

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/nlopes/slack"
	"github.com/rs/zerolog/log"
)

var tunnelOnTime time.Time
var tunnelIdleSince time.Time
var maxTunnelIdleTime = float64(5 * 60) // 5 mins in seconds
var piHostKey = "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBPUURSw9LFDq9q4eI1nTnfNgtK4XZXlA7nhmJfR+NDkJP6Lgv6DRGPL2zJ+drQP7SuZR1uPxsRH4xbZFsNdfhoM="

func homeAndInternetIPsDoNotMatch(tunnelIP string) bool {
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
		RegionName  string `json:"region_name"`
	}
	var jsonRes IPInfoResponse

	select {
	case res := <-results:
		if res != "" {
			err := json.Unmarshal([]byte(res), &jsonRes)
			if err != nil {
				log.Printf("unable to parse JSON string (%v)\n%s\n", err, res)
			} else {
				log.Printf("ipleak.net: %v\n", jsonRes)
			}

			// We're not in Kansas anymore + using tunnel IP for Internet
			if jsonRes.RegionName == "Land Berlin" && jsonRes.IP == tunnelIP {
				resultsDig := make(chan string, 10)
				timeoutDig := time.After(10 * time.Second)
				// ensure home.ackerson.de is DIFFERENT than PI IP address!
				go func() {
					cmd := "dig " + vpnGateway + " A +short"
					log.Printf("%s\n", cmd)
					details := RemoteCmd{Host: raspberryPIIP, Cmd: cmd}

					remoteResult := executeRemoteCmd(details)

					tunnelIdleSince = time.Now()
					resultsDig <- remoteResult.stdout
				}()
				select {
				case resComp := <-resultsDig:
					fmt.Println("dig results: " + resComp)
					lines := strings.Split(resComp, "\n")
					// IPv4 address of home.ackerson.de doesn't match Pi's
					if lines[1] != jsonRes.IP {
						return true
					}
				case <-timeoutDig:
					fmt.Println("Timed out on dig " + vpnGateway + "!")
				}
			}
		}
	case <-timeout:
		fmt.Println("Timed out on curl ipleak.net!")
	}

	return false
}

func inspectVPNConnection() map[string]string {
	results := make(chan string, 10)
	timeout := time.After(10 * time.Second)
	go func() {
		cmd := "sudo ipsec status | grep -A 2 ESTABLISHED"
		details := RemoteCmd{Host: raspberryPIIP, Cmd: cmd}

		remoteResult := executeRemoteCmd(details)

		tunnelIdleSince = time.Now()
		results <- remoteResult.stdout
	}()

	select {
	case res := <-results:
		if res != "" {
			/* look for 1) ESTABLISHED "ago" 2) ...X.Y.Z[<endpointDNS>] 3) internalIP/32 ===
			   proton[34]: ESTABLISHED 89 minutes ago, 192.168.178.59[192.168.178.59]...37.120.217.164[de-14.protonvpn.com]
			   proton{811}:  INSTALLED, TUNNEL, reqid 1, ESP in UDP SPIs: c147cfa6_i c8f7804c_o
			   proton{811}:  10.6.4.224/32 === 0.0.0.0/0
			*/
			re := regexp.MustCompile(`(?s)ESTABLISHED (?P<time>[0-9]+\s\w+)\sago.*\.\.\.(?P<endpointIP>.*)\[(?P<endpointDNS>.*)].*:\s+(?P<internalIP>.*)\/32\s===.*`)
			matches := re.FindAllStringSubmatch(res, -1)
			names := re.SubexpNames()

			m := map[string]string{}
			for i, n := range matches[0] {
				m[names[i]] = n
			}

			if len(m) < 1 {
				cmd := "sudo ipsec restart"
				details := RemoteCmd{Host: raspberryPIIP, Cmd: cmd}

				remoteResult := executeRemoteCmd(details)
				fmt.Println("restarting VPN" + remoteResult.stdout)
			}

			return m
		}
	case <-timeout:
		fmt.Println("Timed out on ipsec status")
	}
	return map[string]string{}
}

// VpnPiTunnelChecks ensures good VPN connection
func VpnPiTunnelChecks(userCall bool) string {
	tunnelIP := ""
	response := ":protonvpn: VPN: DOWN :rotating_light:"

	vpnTunnelSpecs := inspectVPNConnection()
	log.Printf("Using VPN server: %s\n", vpnTunnelSpecs["endpointDNS"])
	if len(vpnTunnelSpecs) > 0 {
		tunnelIP = vpnTunnelSpecs["endpointIP"]
	}

	if homeAndInternetIPsDoNotMatch(tunnelIP) &&
		nftablesUseVPNTunnel(tunnelIP, vpnTunnelSpecs["internalIP"]) {
		response = ":protonvpn: VPN: UP @ " + tunnelIP +
			" for " + vpnTunnelSpecs["time"] + " (using " +
			vpnTunnelSpecs["endpointDNS"] + ")"
	}

	if !userCall {
		customEvent := slack.RTMEvent{Type: "VpnPiTunnelChecks", Data: response}
		rtm.IncomingEvents <- customEvent
	}

	return response
}

func nftablesUseVPNTunnel(tunnelIP string, internalIP string) bool {
	resultsNFTables := make(chan string, 10)
	timeoutNFTables := time.After(5 * time.Second)
	go func() {
		cmd := "sudo nft list ruleset"
		details := RemoteCmd{Host: raspberryPIIP, Cmd: cmd}

		remoteResult := executeRemoteCmd(details)

		tunnelIdleSince = time.Now()
		resultsNFTables <- remoteResult.stdout
	}()

	select {
	case resNFTables := <-resultsNFTables:
		if strings.Contains(resNFTables, "ip daddr "+tunnelIP) &&
			strings.Contains(resNFTables, "ip saddr "+tunnelIP) &&
			strings.Contains(resNFTables, "oifname \"eth0\" ip saddr "+internalIP) &&
			strings.Contains(resNFTables, "iifname \"eth0\" ip daddr "+internalIP) {
			return true
		}

		cmd := "sudo nft -f /etc/nftables.conf && sudo ipsec restart && sudo service transmission-daemon restart"
		details := RemoteCmd{Host: raspberryPIIP, Cmd: cmd}

		remoteResult := executeRemoteCmd(details)
		fmt.Println("reset nftables, VPN & transmission: " + remoteResult.stdout)

	case <-timeoutNFTables:
		fmt.Println("Timed out on `sudo nft list ruleset`!")
	}

	return false
}
