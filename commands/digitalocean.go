package commands

import (
	"fmt"
	"strconv"

	"github.com/danackerson/digitalocean/common"
	"github.com/nlopes/slack"
)

// ListDODroplets is now commented
func ListDODroplets(userCall bool) string {
	doDropletInfoSite := "https://cloud.digitalocean.com/droplets/"
	response := ""

	client := common.PrepareDigitalOceanLogin()

	droplets, err := common.DropletList(client)
	if err != nil {
		response = fmt.Sprintf("Failed to list droplets: %s", err)
	} else {
		for _, droplet := range droplets {
			ipv4, _ := droplet.PublicIPv4()
			addr := doDropletInfoSite + strconv.Itoa(droplet.ID)
			response += fmt.Sprintf(":do_droplet: <%s|%s> (%s) [ID: %d]\n", addr, droplet.Name, ipv4, droplet.ID)
		}
	}

	if !userCall {
		rtm.IncomingEvents <- slack.RTMEvent{Type: "ListDODroplets", Data: response}
	}

	return response
}
