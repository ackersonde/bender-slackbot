package commands

import (
	"fmt"
	"strconv"

	"github.com/danackerson/digitalocean/common"
)

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
