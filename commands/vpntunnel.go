package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/ackersonde/bender-slackbot/structures"
	"github.com/slack-go/slack"
)

var vpnLogicalsURI = "https://api.protonmail.ch/vpn/logicals"
var maxVPNServerLoad = 80

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
		cmd := "docker exec vpnission ipsec status | grep -A 2 ESTABLISHED"
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
			/* look for 1) ESTABLISHED "ago" 2) ...X.Y.Z[<endpointDNS>] 3) internalIP/32 ===
			   proton[34]: ESTABLISHED 89 minutes ago, 192.168.178.59[192.168.178.59]...37.120.217.164[de-14.protonvpn.com]
			   proton{811}:  INSTALLED, TUNNEL, reqid 1, ESP in UDP SPIs: c147cfa6_i c8f7804c_o
			   proton{811}:  10.6.4.224/32 === 0.0.0.0/0
			*/
			re := regexp.MustCompile(`(?s)ESTABLISHED (?P<time>[0-9]+\s\w+)\sago.*\.\.\.(?P<endpointIP>.*)\[(?P<endpointDNS>.*)].*:\s+(?P<internalIP>.*)\/32\s===.*`)
			matches := re.FindAllStringSubmatch(res, -1)
			names := re.SubexpNames()

			m := map[string]string{}
			if len(matches) > 0 {
				for i, n := range matches[0] {
					m[names[i]] = n
				}
			} else {
				Logger.Printf("ERR: IPSec down")
			}

			return m
		}
	case <-timeout:
		Logger.Printf("ERR: Timed out on IPSec status")
	}
	return map[string]string{}
}

func findBestVPNServer(vpnCountry string) structures.LogicalServer {
	protonVPNServers := new(structures.ProtonVPNServers)
	protonVPNServersResp, err := http.Get(vpnLogicalsURI)
	if err != nil {
		Logger.Printf("protonVPN API ERR: %s\n", err)
	} else {
		defer protonVPNServersResp.Body.Close()
		protonVPNServersJSON, err2 := ioutil.ReadAll(protonVPNServersResp.Body)
		if err2 != nil {
			Logger.Printf("protonVPN ERR2: %s\n", err2)
		}
		json.Unmarshal([]byte(protonVPNServersJSON), &protonVPNServers)
	}

	// we're only interested in premium VPN servers from one country
	i := 0
	for k, x := range protonVPNServers.LogicalServers {
		if protonVPNServers.LogicalServers[k].EntryCountry == vpnCountry &&
			protonVPNServers.LogicalServers[k].Tier >= 2 {
			protonVPNServers.LogicalServers[i] = x
			i++
		} else {
			continue
		}
	}
	protonVPNServers.LogicalServers = protonVPNServers.LogicalServers[:i]

	// order servers by highest score
	sort.Slice(protonVPNServers.LogicalServers, func(i, j int) bool {
		return protonVPNServers.LogicalServers[i].Score > protonVPNServers.LogicalServers[j].Score
	})

	var bestServer structures.LogicalServer

	// suggest highest scoring VPN server with load < maxVPNServerLoad
	for k := range protonVPNServers.LogicalServers {
		if protonVPNServers.LogicalServers[k].Load < maxVPNServerLoad {
			bestServer = protonVPNServers.LogicalServers[k]
			break
		}
	}

	return bestServer
}

// ChangeToFastestVPNServer on cronjob call
func ChangeToFastestVPNServer(vpnCountry string) {
	response := "Failed auto VPN update"

	bestVPNServer := findBestVPNServer(vpnCountry)
	response = updateVpnPiTunnel(bestVPNServer.Domain)

	api.PostMessage(SlackReportChannel, slack.MsgOptionText(response, false),
		slack.MsgOptionAsUser(true))
}

// VpnPiTunnelChecks ensures correct VPN connection
func VpnPiTunnelChecks(vpnCountry string) string {
	ipsecVersion := executeRemoteCmd(
		"docker exec vpnission ipsec --version | head -n 1",
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

	bestVPNServer := findBestVPNServer(vpnCountry)
	response = response + "\n\nBest in " + vpnCountry +
		fmt.Sprintf(" %d%% ", bestVPNServer.Load)

	if strings.Contains(response, ":protonvpn: VPN: DOWN") {
		response = ipsecVersion.Stdout + "VPN was DOWN! Restarting...\n" +
			updateVpnPiTunnel(bestVPNServer.Domain)
	} else {
		// check if we're already using the Best server
		remoteResult := executeRemoteCmd("host "+bestVPNServer.Domain,
			structures.VPNPIRemoteConnectConfig)
		if remoteResult.Err != nil {
			response += "Couldn't get IP of " + bestVPNServer.Domain
		} else {
			response += remoteResult.Stdout
		}
	}

	return response
}

func updateVpnPiTunnel(vpnServerDomain string) string {
	response := "Failed changing :protonvpn: to " + vpnServerDomain

	stopVPNCmd := `docker rm -f vpnission && `
	startVPNCmd := `docker run --env-file .config/vpnission.env.list -d \
        --restart=always --name vpnission --cap-add NET_ADMIN -p9091:9091 -p51413:51413 \
        -v /mnt/usb4TB/DLNA/torrents:/mnt/torrents \
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
