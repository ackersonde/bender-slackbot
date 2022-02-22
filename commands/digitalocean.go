package commands

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/ackersonde/bender-slackbot/structures"
	"github.com/ackersonde/digitaloceans/common"
	"github.com/ackersonde/hetzner/hetznercloud"
	"github.com/slack-go/slack"
)

func fetchExtraHetznerFirewallRules(homeIPv6Prefix string) []string {
	var extraRules []string
	log.Printf("HomeIPv6Prefix: %s\n", homeIPv6Prefix)

	sshFWRules := hetznercloud.GetSSHFirewallRules()
	for _, rule := range sshFWRules {
		if strings.TrimSpace(rule) != homeIPv6Prefix {
			extraRules = append(extraRules, rule)
		}
	}

	return extraRules
}

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
// cloud instances and home network, ensuring minimal connectivity
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
		response += "allowed from " + homeIPv6Prefix + " :house:"
	}

	response += "\n\n:htz_server: "
	extras = fetchExtraHetznerFirewallRules(homeIPv6Prefix)
	if len(extras) > 0 {
		response += "<https://console.hetzner.cloud/projects/" + os.Getenv("CTX_HETZNER_PROJECT") +
			"/firewalls/" + os.Getenv("CTX_HETZNER_FIREWALL") + "/rules|open to> -> " +
			strings.Join(extras, ", "+":rotating_light")
	} else {
		response += "allowed from " + homeIPv6Prefix + " :house:"
	}

	response += "\n\n:house: "

	domainIPv6 := getIPv6forHostname("ackerson.de")
	homeFirewallRules := checkHomeFirewallSettings(domainIPv6, homeIPv6Prefix)
	if len(homeFirewallRules) > 0 {
		response += "opened on -> " + strings.Join(homeFirewallRules, "\n") + " :rotating_light:"
	} else {
		response += "allowed from " + domainIPv6 + " :do_droplet:"
	}

	return response
}
