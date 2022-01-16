package commands

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/ackersonde/bender-slackbot/structures"
	"github.com/ackersonde/digitaloceans/common"
	"github.com/slack-go/slack"
)

func fetchExtraDOsshFirewallRules(homeIPv6Prefix string) []string {
	var extraRules []string
	log.Printf("HomeIPv6Prefix: %s\n", homeIPv6Prefix)

	sshFWRules := common.GetSSHFirewallRules()
	for _, rule := range sshFWRules {
		if strings.TrimSpace(rule) != homeIPv6Prefix {
			extraRules = append(extraRules, rule)
		}
	}

	return extraRules
}

// ListDODroplets is now commented
func ListDODroplets() string {
	doDropletInfoSite := "https://cloud.digitalocean.com/droplets/"
	response := ""

	client := common.PrepareDigitalOceanLogin()

	droplets, err := common.DropletList(client)
	if err != nil {
		response = fmt.Sprintf("Failed to list droplets: %s", err)
	} else {
		response = fmt.Sprintf("Found %d droplet(s):", len(droplets))
		for _, droplet := range droplets {
			ipv4, _ := droplet.PublicIPv4()
			addr := doDropletInfoSite + strconv.Itoa(droplet.ID)
			response += fmt.Sprintf(":do_droplet: <%s|%s> (%s) [ID: %d]\n", addr, droplet.Name, ipv4, droplet.ID)
		}
	}

	return response
}

// DisplayFirewallRules for daily cron
func DisplayFirewallRules() {
	api.PostMessage(SlackReportChannel, slack.MsgOptionText(checkFirewallRules(), false),
		slack.MsgOptionAsUser(true))
}

// checkFirewallRules does a cross check of SSH access between
// digital ocean instance and home networks, ensuring minimal connectivity
func checkFirewallRules() string {
	executeRemoteCmd("wakeonlan 2c:f0:5d:5e:84:43", structures.PI4RemoteConnectConfig)
	homeIPv6Prefix := fetchHomeIPv6Prefix()
	extras := fetchExtraDOsshFirewallRules(homeIPv6Prefix)

	response := ":do_droplet: "
	if len(extras) > 0 {
		response += "<https://cloud.digitalocean.com/networking/firewalls/" +
			os.Getenv("CTX_DIGITALOCEAN_FIREWALL") + "/rules|open to> -> " +
			strings.Join(extras, ", ") + " :rotating_light:"
	} else {
		response += "secured for " + homeIPv6Prefix + " :white_check_mark:"
	}

	response += "\n\n:house: "

	domainIPv6 := getIPv6forHostname("ackerson.de")
	homeFirewallRules := checkHomeFirewallSettings(domainIPv6, homeIPv6Prefix)
	if len(homeFirewallRules) > 0 {
		response += "opened on -> " + strings.Join(homeFirewallRules, "\n") + " :rotating_light:"
	} else {
		response += "secured for " + domainIPv6 + " :white_check_mark:"
	}

	return response
}
