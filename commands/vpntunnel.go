package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/danackerson/bender-slackbot/structures"
	"github.com/nlopes/slack"
	"github.com/rs/zerolog/log"
)

var vpnLogicalsURI = "https://api.protonmail.ch/vpn/logicals"
var maxVPNServerLoad = 80
var tunnelOnTime time.Time
var tunnelIdleSince time.Time
var maxTunnelIdleTime = float64(5 * 60) // 5 mins in seconds
var vpnPIHostKey = "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBCfXJ+mvHXs+t0+nF8JATxgMEwNngy6JCOVn1bEjsjsMylZsMejouArUrNKcnyPZ+vTvljlR7CaC6X9fbUtdxs0="
var pi4HostKey = "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBC9wGQXT5zifmoWRaLeDrf/j98ShzZ29CilfVUVtSeKJp1k2uh8pMM/NTiG9FQQmitEIZXdwlcl2+Uj8YD21sAI="
var raspi3HostKey = "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBD7p4FZyTPgywRBJ2ADL/i2igJ/N+3G8odFL3or3Ck77CVBnri8ZxO8+34/Rl/eGgt9qhp0vm7eTB4nE0C2m/Ro="

func homeAndInternetIPsDoNotMatch(tunnelIP string) bool {
	results := make(chan string, 10)
	timeout := time.After(10 * time.Second)
	go func() {
		// get both ipv4+ipv6 internet addresses
		cmd := "curl https://ipleak.net/json/"
		details := RemoteCmd{Host: raspberryPIIP, Cmd: cmd}

		remoteResult := executeRemoteCmd(details, retrieveClientConfig(vpnPIRemoteConnectConfig))

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
			if jsonRes.IP == tunnelIP {
				resultsDig := make(chan string, 10)
				timeoutDig := time.After(10 * time.Second)
				// ensure home.ackerson.de is DIFFERENT than PI IP address!
				go func() {
					cmd := "dig " + vpnGateway + " A +short"
					log.Printf("%s\n", cmd)
					details := RemoteCmd{Host: raspberryPIIP, Cmd: cmd}

					remoteResult := executeRemoteCmd(details, retrieveClientConfig(vpnPIRemoteConnectConfig))

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

		remoteResult := executeRemoteCmd(details, retrieveClientConfig(vpnPIRemoteConnectConfig))

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

				remoteResult := executeRemoteCmd(details, retrieveClientConfig(vpnPIRemoteConnectConfig))
				fmt.Println("restarting VPN" + remoteResult.stdout)
			}

			return m
		}
	case <-timeout:
		fmt.Println("Timed out on ipsec status")
	}
	return map[string]string{}
}

func findBestVPNServer(vpnCountry string) structures.LogicalServer {
	protonVPNServers := new(structures.ProtonVPNServers)
	protonVPNServersResp, err := http.Get(vpnLogicalsURI)
	if err != nil {
		log.Printf("protonVPN API ERR: %s\n", err)
	} else {
		defer protonVPNServersResp.Body.Close()
		protonVPNServersJSON, err2 := ioutil.ReadAll(protonVPNServersResp.Body)
		if err2 != nil {
			log.Printf("protonVPN ERR2: %s\n", err2)
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
func ChangeToFastestVPNServer(vpnCountry string, userCall bool) string {
	response := "Failed auto VPN update"

	bestVPNServer := findBestVPNServer(vpnCountry)
	response = updateVpnPiTunnel(bestVPNServer.Domain)
	if !userCall {
		customEvent := slack.RTMEvent{Type: "ChangeToFastestVPNServer", Data: response}
		rtm.IncomingEvents <- customEvent
	}

	return response
}

// VpnPiTunnelChecks ensures good VPN connection
func VpnPiTunnelChecks(vpnCountry string, userCall bool) string {
	tunnelIP := ""
	response := ":protonvpn: VPN: DOWN :rotating_light:"

	vpnTunnelSpecs := inspectVPNConnection()
	log.Printf("Using VPN server: %s\n", vpnTunnelSpecs["endpointDNS"])
	if len(vpnTunnelSpecs) > 0 {
		tunnelIP = vpnTunnelSpecs["endpointIP"]
	}

	if homeAndInternetIPsDoNotMatch(tunnelIP) {
		// TODO && transmission-daemon:bind-address != internal VPN IP (10.6.*.*)
		response = ":protonvpn: VPN: UP @ " + tunnelIP +
			" for " + vpnTunnelSpecs["time"] + " (using " +
			vpnTunnelSpecs["endpointDNS"] + ")"
	}

	bestVPNServer := findBestVPNServer(vpnCountry)
	response = response + "\n\nBest VPN server in " + vpnCountry + " => " +
		fmt.Sprintf("Tier:%d Load:%d Score:%f %s\n",
			bestVPNServer.Tier,
			bestVPNServer.Load,
			bestVPNServer.Score,
			bestVPNServer.Domain)

	if !userCall {
		customEvent := slack.RTMEvent{Type: "VpnPiTunnelChecks", Data: response}
		rtm.IncomingEvents <- customEvent
	}

	return response
}

func updateVpnPiTunnel(vpnServerDomain string) string {
	if !strings.HasSuffix(vpnServerDomain, ".protonvpn.com") {
		vpnServerDomain = vpnServerDomain + ".protonvpn.com"
	}
	response := "Failed to change VPN server to " + vpnServerDomain

	sedCmd := `sudo sed -rie 's@[A-Za-z]{2}-[0-9]{2}\.protonvpn\.com@` + vpnServerDomain + `@g' `
	cmd := sedCmd + "/etc/ipsec.conf"
	details := RemoteCmd{Host: raspberryPIIP, Cmd: cmd}

	remoteResult := executeRemoteCmd(details, retrieveClientConfig(vpnPIRemoteConnectConfig))
	// TODO: stderr often doesn't have real errors :(
	if remoteResult.err == nil {
		cmd = "ip address | grep \"10\\.\"" // search for internal VPN ip
		details = RemoteCmd{Host: raspberryPIIP, Cmd: cmd}

		remoteResult = executeRemoteCmd(details, retrieveClientConfig(vpnPIRemoteConnectConfig))
		log.Printf("internal VPN IP: %s", remoteResult.stdout)
		if remoteResult.err == nil {
			// files updated - now restart everything
			cmd = `sudo service transmission-daemon stop &&
			sudo ipsec update &&
			sudo ipsec restart && sudo service transmission-daemon start`
			details = RemoteCmd{Host: raspberryPIIP, Cmd: cmd}

			// TODO: need to change Transmission-Daemon bind-address-ipv4
			// ip address | grep "10\." -> inet "10.6.4.235/32" scope global eth0
			// sudo vi ~/.config/transmission-daemon/settings.json -> "bind-address-ipv4": "10.6.4.235",

			remoteResult = executeRemoteCmd(details, retrieveClientConfig(vpnPIRemoteConnectConfig))
			if remoteResult.err == nil {
				response = "Changed VPN server to " + vpnServerDomain
			}
		}
	}

	if remoteResult.err != nil {
		response += "(" + remoteResult.err.Error() + ")"
	}

	return response
}
