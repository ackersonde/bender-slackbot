package commands

import (
	"os"
	"strings"

	"github.com/ackersonde/bender-slackbot/structures"
	"github.com/ackersonde/hetzner_home/hetznercloud"
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
	response := ""

	extras := fetchExtraHetznerFirewallRules(homeIPv6Prefix)
	if len(extras) > 0 {
		response += ":htz_server: <https://console.hetzner.cloud/projects/" + os.Getenv("HETZNER_PROJECT") +
			"/firewalls/" + os.Getenv("HETZNER_FIREWALL") + "/rules|open to> -> " +
			"`" + strings.Join(extras, "`, `") + "`" + " :rotating_light:\n\n"
	} else if manuallyCalled {
		response += ":htz_server: allowed from `" + homeIPv6Prefix + "` :house:\n\n"
	}

	domainIPv6 := getIPv6forHostname("ackerson.de")
	homeFirewallRules := checkHomeFirewallSettings(domainIPv6, homeIPv6Prefix)
	if len(homeFirewallRules) > 0 {
		response += ":house: opened on -> `" + strings.Join(homeFirewallRules, "`, `") + "`" + " :rotating_light:"
	} else if manuallyCalled {
		response += ":house: allowed from `" + domainIPv6 + "` :htz_server:"
	}

	return strings.TrimSuffix(response, "\n")
}
