package commands

// forked from https://github.com/jasonrhansen/piratebay
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/danackerson/bender-slackbot/structures"
)

// Category is the type of torrent to search for.
type Category uint16

const (
	// Audio is the Category used to search for audio torrents.
	Audio Category = 100
	// Video is the Category used to search for video torrents.
	Video Category = 200
	// HDMovies is the Category used to search for HD movie torrents.
	HDMovies Category = 207
	// Applications is the Category used to search for applications torrents.
	Applications Category = 300
	// Games is the Category used to search for games torrents.
	Games Category = 400
)

var proxies = []string{"tpb.cool", "piratebay.tech", "thepiratebay.fail", "piratebay.icu", "thepirate.host"}

func searchProxy(url string) *structures.Torrents {
	for i, proxy := range proxies {
		uri := "https://" + proxy + url
		log.Printf("torq try #%d: %s ...\n", i, uri)
		req, err := http.NewRequest("GET", uri, nil)
		if err != nil {
			log.Printf("http.NewRequest() failed with '%s'\n", err)
			continue
		}

		// create a context indicating 5s timeout
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*5000)
		defer cancel()
		req = req.WithContext(ctx)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("%s failed with:\n'%s'\n", proxy, err)
			continue
		}
		if resp != nil {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Printf("failed to parse JSON: %s", err.Error())
				continue
			}

			if err == nil && body != nil {
				torrents := getTorrentsFromJSON([]byte(body))
				if torrents != nil {
					log.Printf("%s%s succeeded!", proxy, url)
					return torrents
				}
			}
		}
	}

	return nil
}

// SearchFor is now commented
func SearchFor(term string, cat Category) (*structures.Torrents, string) {
	response := ""

	var torrents *structures.Torrents
	torrents, err := search(term, cat)
	found := 0
	if err == nil {
		for i, t := range *torrents {
			seeders, _ := strconv.Atoi(t.Seeders)
			if seeders > 10 {
				found++
				size, _ := strconv.Atoi(t.Size)
				humanSize := float64(size / (1024 * 1024))
				sizeSuffix := fmt.Sprintf("*%.0f MiB*", humanSize)
				if humanSize > 999 {
					humanSize = humanSize / 1024
					sizeSuffix = fmt.Sprintf("*%.1f GiB*", humanSize)
				}

				magnetLink := fmt.Sprintf("magnet/?xt=urn:btih:%s", t.InfoHash)
				response += fmt.Sprintf("%d: <http://%s|%s> Seeds:%d %s\n", i, magnetLink, t.Name, seeders, sizeSuffix)
			}
		}
	} else {
		response = "PB seems to be offline: " + fmt.Sprintf("%v", err)
	}

	if found < 1 {
		response = "Unable to find torrents with enough Seeders for '" + term + "'"
	}

	return torrents, response
}

// search returns the torrents found with the given search string and categories.
func search(query string, cats ...Category) (*structures.Torrents, error) {
	var torrents *structures.Torrents

	if query != "" {
		if len(cats) == 0 {
			cats = []Category{0}
		}

		var catStr string
		for i, c := range cats {
			if i != 0 {
				catStr += ","
			}
			catStr += strconv.Itoa(int(c))
		}
		if catStr == "" {
			catStr = "0"
		}

		searchStringURL := "/api?url=/q.php?q=" + url.QueryEscape(query) + "&cat=" + catStr
		torrents = searchProxy(searchStringURL)
	} else {
		torrents = searchProxy("/api?url=/q.php?q=category%3A207")
	}

	if torrents == nil {
		return nil, errors.New("unable to contact any PB Proxy...try again later")
	}

	return torrents, nil
}

func getTorrentsFromJSON(jsonObject []byte) *structures.Torrents {
	var s = new(structures.Torrents)
	err := json.Unmarshal(jsonObject, &s)
	if err != nil {
		fmt.Println("whoops:", err)
	}

	return s
}
