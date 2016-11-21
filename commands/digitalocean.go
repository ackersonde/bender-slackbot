package commands

import (
	"fmt"
	"os"
	"strconv"

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

// ListDODroplets is now commented
func ListDODroplets() string {
	doDropletInfoSite := "https://cloud.digitalocean.com/droplets/"
	doPersonalAccessToken := os.Getenv("digitalOceanToken")
	response := ""

	tokenSource := &TokenSource{
		AccessToken: doPersonalAccessToken,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	droplets, err := DropletList(client)
	if err != nil {
		response = fmt.Sprintf("Failed to list droplets: %s", err)
	} else {
		for _, droplet := range droplets {
			ipv4, _ := droplet.PublicIPv4()
			addr := doDropletInfoSite + strconv.Itoa(droplet.ID)
			response += fmt.Sprintf(":do_droplet: <%s|%s> (%s)\n", addr, droplet.Name, ipv4)
		}
	}

	customEvent := slack.RTMEvent{Type: "ListDODroplets", Data: response}
	rtm.IncomingEvents <- customEvent
	return response
}

// DropletList is now commented
func DropletList(client *godo.Client) ([]godo.Droplet, error) {
	// create a list to hold our droplets
	list := []godo.Droplet{}

	// create options. initially, these will be blank
	opt := &godo.ListOptions{}
	for {
		droplets, resp, err := client.Droplets.List(opt)
		if err != nil {
			return nil, err
		}

		// append the current page's droplets to our list
		for _, d := range droplets {
			list = append(list, d)
		}

		// if we are at the last page, break out the for loop
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, err
		}

		// set the page we want for the next request
		opt.Page = page + 1
	}

	return list, nil
}
