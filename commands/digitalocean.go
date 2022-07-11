package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/ackersonde/bender-slackbot/structures"
	"github.com/ackersonde/digitaloceans/common"
	"github.com/ackersonde/hetzner_vault/hetznercloud"
	"github.com/slack-go/slack"
)

func fetchExtraHetznerFirewallRules(homeIPv6Prefix string) []string {
	var extraRules []string

	sshFWRules := hetznercloud.GetSSHFirewallRules()
	for _, rule := range sshFWRules {
		if strings.TrimSpace(rule) != homeIPv6Prefix {
			Logger.Printf("%s doesn't MATCH %s\n", rule, homeIPv6Prefix)
			extraRules = append(extraRules, rule)
		}
	}

	return extraRules
}

func fetchExtraDOsshFirewallRules(homeIPv6Prefix string) []string {
	var extraRules []string
	Logger.Printf("HomeIPv6Prefix: %s\n", homeIPv6Prefix)

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
		response = fmt.Sprintf("Found %d droplet(s) at :do_droplet::\n", len(droplets))
		for _, droplet := range droplets {
			ipv6, _ := droplet.PublicIPv6()
			addr := doDropletInfoSite + strconv.Itoa(droplet.ID)
			response += fmt.Sprintf("ID %d: <%s|%s> [%s] @ %s => %s\n",
				droplet.ID, addr, droplet.Name, ipv6, droplet.Created, droplet.Status)
		}
	}

	out, _ := exec.Command("uptime").Output()
	response += string(out)
	out, _ = exec.Command("uname", "-a").Output()
	response += string(out)

	return response
}

// DisplayFirewallRules for daily cron
func DisplayFirewallRules() {
	openFirewallRules := checkFirewallRules(false)
	if openFirewallRules != "" {
		api.PostMessage(SlackReportChannel, slack.MsgOptionText(openFirewallRules, false),
			slack.MsgOptionAsUser(true))
	}
}

// checkFirewallRules does a cross check of SSH access between
// cloud instances and home network, ensuring minimal connectivity
func checkFirewallRules(manuallyCalled bool) string {
	executeRemoteCmd("wakeonlan 2c:f0:5d:5e:84:43", structures.PI4RemoteConnectConfig)
	homeIPv6Prefix := fetchHomeIPv6Prefix()
	extras := fetchExtraDOsshFirewallRules(homeIPv6Prefix)

	response := ""
	if len(extras) > 0 {
		response += ":do_droplet: <https://cloud.digitalocean.com/networking/firewalls/" +
			os.Getenv("CTX_DIGITALOCEAN_FIREWALL") + "/rules|open to> -> " +
			"`" + strings.Join(extras, "`, `") + "`" + " :rotating_light:\n\n"
	} else if manuallyCalled {
		response += ":do_droplet: allowed from `" + homeIPv6Prefix + "` :house:\n\n"
	}

	extras = fetchExtraHetznerFirewallRules(homeIPv6Prefix)
	if len(extras) > 0 {
		response += ":htz_server: <https://console.hetzner.cloud/projects/" + os.Getenv("ORG_HETZNER_PROJECT") +
			"/firewalls/" + os.Getenv("CTX_HETZNER_FIREWALL") + "/rules|open to> -> " +
			"`" + strings.Join(extras, "`, `") + "`" + " :rotating_light:\n\n"
	} else if manuallyCalled {
		response += ":htz_server: allowed from `" + homeIPv6Prefix + "` :house:\n\n"
	}

	domainIPv6 := getIPv6forHostname("ackerson.de")
	homeFirewallRules := checkHomeFirewallSettings(domainIPv6, homeIPv6Prefix)
	if len(homeFirewallRules) > 0 {
		response += ":house: opened on -> `" + strings.Join(homeFirewallRules, "`, `") + "`" + " :rotating_light:"
	} else if manuallyCalled {
		response += ":house: allowed from `" + domainIPv6 + "` :do_droplet:"
	}

	return strings.TrimSuffix(response, "\n")
}
