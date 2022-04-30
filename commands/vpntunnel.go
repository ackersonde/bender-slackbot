package commands

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/ackersonde/bender-slackbot/structures"
	"github.com/slack-go/slack"
)

func homeAndInternetIPsDoNotMatch(tunnelIP string) bool {
	results := make(chan string, 10)
	timeout := time.After(10 * time.Second)
	ipCheckHost := "https://ipv4.icanhazip.com" // TODO: update to ipv6 once VPN supports it

	go func() {
		cmd := "docker exec vpnission curl -s " + ipCheckHost
		remoteResult := executeRemoteCmd(cmd, structures.VPNPIRemoteConnectConfig)

		results <- strings.TrimSuffix(remoteResult.Stdout, "\n")
	}()

	select {
	case res := <-results:
		if res != "" {
			// We're not in Kansas anymore + using tunnel IP for Internet
			resultsDig := make(chan string, 10)
			timeoutDig := time.After(10 * time.Second)
			// ensure home.ackerson.de is DIFFERENT than PI IP address!
			go func() {
				cmd := "curl -s " + ipCheckHost
				remoteResult := executeRemoteCmd(cmd, structures.VPNPIRemoteConnectConfig)

				resultsDig <- remoteResult.Stdout
			}()
			select {
			case resComp := <-resultsDig:
				Logger.Printf("on %s `curl -s %s` = %s", os.Getenv("SLAVE_HOSTNAME"), ipCheckHost, resComp)
				// IPv4 address of home.ackerson.de doesn't match Pi's
				if resComp != res {
					return true
				}
			case <-timeoutDig:
				Logger.Printf("Time out on curl -s %s", ipCheckHost)
			}
		}
	case <-timeout:
		Logger.Printf("Timeout on curl %s", ipCheckHost)
	}

	return false
}

func inspectVPNConnection() map[string]string {
	resultsChannel := make(chan string, 10)
	timeout := time.After(10 * time.Second)
	go func() {
		cmd := "docker exec vpnission wg show | grep endpoint"
		remoteResult := executeRemoteCmd(cmd, structures.VPNPIRemoteConnectConfig)

		result := ""
		if remoteResult.Stdout == "" && remoteResult.Stderr != "" {
			result += remoteResult.Stderr
		} else {
			result += remoteResult.Stdout
		}
		resultsChannel <- result
	}()

	select {
	case res := <-resultsChannel:
		if res != "" {
			// look for `endpoint: <IP-Addr>:51820`
			re := regexp.MustCompile(`endpoint: (?P<endpointIP>.*):51820`)
			matches := re.FindAllStringSubmatch(res, -1)
			names := re.SubexpNames()

			m := map[string]string{}
			if len(matches) > 0 {
				for i, n := range matches[0] {
					m[names[i]] = n
				}
			} else {
				Logger.Printf("ERR: Wireguard down")
			}

			return m
		}
	case <-timeout:
		Logger.Printf("ERR: Timed out on IPSec status")
	}
	return map[string]string{}
}

func ChangeToNextWireguardServer(vpnCountry string) {
	response := "Failed auto VPN update"

	response = updateVpnPiTunnel("NL_28")

	api.PostMessage(SlackReportChannel, slack.MsgOptionText(response, false),
		slack.MsgOptionAsUser(true))
}

// VpnPiTunnelChecks ensures correct VPN connection
func VpnPiTunnelChecks(vpnCountry string) string {
	ipsecVersion := executeRemoteCmd(
		"docker exec vpnission wg --version",
		structures.VPNPIRemoteConnectConfig)
	response := ipsecVersion.Stdout + ":protonvpn: VPN: DOWN :rotating_light:"

	vpnTunnelSpecs := inspectVPNConnection()
	if len(vpnTunnelSpecs) > 0 {
		Logger.Printf("Using VPN server: %s\n", vpnTunnelSpecs["endpointDNS"])
		response += " with " + vpnTunnelSpecs["time"] + " (using " +
			vpnTunnelSpecs["endpointDNS"] + ")"

		if homeAndInternetIPsDoNotMatch(vpnTunnelSpecs["endpointIP"]) &&
			transmissionSettingsAreSane(vpnTunnelSpecs["internalIP"]) {
			response = ipsecVersion.Stdout + ":protonvpn: VPN: UP for " +
				vpnTunnelSpecs["time"] + " (*" +
				vpnTunnelSpecs["endpointDNS"] + "*)"
		}
	}

	if strings.Contains(response, ":protonvpn: VPN: DOWN") {
		response = ipsecVersion.Stdout + "VPN was DOWN! Restarting...\n" +
			updateVpnPiTunnel("NL_88")
	} else {
		// check if we're already using the Best server
		response = "switching to 'NL_88'"
	}

	return response
}

func updateVpnPiTunnel(vpnServerDomain string) string {
	response := "Failed changing :protonvpn: to " + vpnServerDomain

	stopVPNCmd := `docker rm -f vpnission && `
	startVPNCmd := `docker run --env-file .config/vpnission.env.list -d \
        --restart=always --name vpnission --privileged -p9091:9091 -p51413:51413 \
        -v /etc/wireguard:/etc/wireguard -v /mnt/usb4TB/DLNA/torrents:/mnt/torrents \
        danackerson/vpnission ` + vpnServerDomain

	cmd := stopVPNCmd + startVPNCmd

	remoteResult := executeRemoteCmd(cmd, structures.VPNPIRemoteConnectConfig)
	if remoteResult.Err != nil {
		response += fmt.Sprintf("\nErrors on updating protonvpn: %v\n", remoteResult)
	} else {
		response = "Updated :protonvpn: to " + vpnServerDomain
	}

	return response
}
