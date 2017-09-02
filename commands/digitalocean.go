package commands

import (
	"fmt"
	"os"
	"strconv"

	"github.com/danackerson/digitalocean/common"
	"github.com/digitalocean/godo"
	"github.com/nlopes/slack"

	"golang.org/x/oauth2"
)

// TokenSource is now commented
type TokenSource struct {
	AccessToken string
}

// Token is now commented
func (t *TokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}

func prepareDigitalOceanLogin() *godo.Client {
	doPersonalAccessToken := os.Getenv("digitalOceanToken")
	tokenSource := &TokenSource{
		AccessToken: doPersonalAccessToken,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	return godo.NewClient(oauthClient)
}

// DeleteDODroplet more here https://developers.digitalocean.com/documentation/v2/#delete-a-droplet
func DeleteDODroplet(ID int) string {
	var result string

	client := prepareDigitalOceanLogin()

	_, err := client.Droplets.Delete(oauth2.NoContext, ID)
	if err == nil {
		result = "Successfully deleted Droplet `" + strconv.Itoa(ID) + "`"
	} else {
		result = err.Error()
	}

	return result
}

// ListDODroplets is now commented
func ListDODroplets(userCall bool) string {
	doDropletInfoSite := "https://cloud.digitalocean.com/droplets/"
	response := ""

	client := prepareDigitalOceanLogin()

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
