package main

import (
	"fmt"
	"strings"

	"github.com/ackersonde/bender-slackbot/commands"
	"github.com/ackersonde/bender-slackbot/structures"
)

func main2() {
	vpnServerDomain := "de-24.protonvpn.com"

	response := "Failed changing :protonvpn: to " + vpnServerDomain
	sedCmd := `sudo sed -rie 's@[A-Za-z]{2}-[0-9]{2}\.protonvpn\.com@` + vpnServerDomain + `@g' `
	cmd := sedCmd + `/etc/ipsec.conf && sudo ipsec update && sudo ipsec up proton`

	remoteResult := commands.ExecuteRemoteCmd(cmd, structures.VPNPIRemoteConnectConfig)
	if remoteResult.Err == nil ||
		strings.HasSuffix(remoteResult.Err.Error(), "exited with status 7") {
		response = "Updated :protonvpn: to " + vpnServerDomain
	}

	fmt.Println(response)
}
