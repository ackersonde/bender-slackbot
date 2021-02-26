package commands

// forked from https://github.com/jasonrhansen/piratebay
import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/ackersonde/bender-slackbot/structures"
)

var proxies = []string{"tpb.cool", "piratebay.icu", "thepirate.host"}

func searchProxy(url string) []byte {
	var jsonResults []byte

	for i, proxy := range proxies {
		uri := "https://" + proxy + url
		Logger.Printf("torq try #%d: %s ...\n", i, uri)
		req, err := http.NewRequest("GET", uri, nil)
		if err != nil {
			Logger.Printf("http.NewRequest() failed with '%s'\n", err)
			continue
		}

		// create a context indicating 5s timeout
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*5000)
		defer cancel()
		req = req.WithContext(ctx)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			Logger.Printf("%s failed with:\n'%s'\n", proxy, err)
			continue
		}
		if resp != nil {
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil || body == nil {
				Logger.Printf("failed to parse JSON: %s", err.Error())
				continue
			}

			return body
		}
	}

	return jsonResults
}

func parseTop100(jsonResponse []byte) string {
	return top100Response(*getTop100FromJSON(jsonResponse))
}

func parseTorrents(jsonResponse []byte) string {
	return torrentResponse(*getTorrentsFromJSON(jsonResponse))
}

func top100Response(top100 structures.Top100Movies) string {
	var response string

	for i, torrent := range top100 {
		if torrent.Seeders > 10 {
			if torrent.Imdb == nil {
				torrent.Imdb = ""
			}
			response += prepareLink(
				i, torrent.InfoHash, torrent.Name,
				torrent.Seeders, calculateHumanSize(torrent.Size),
				torrent.Imdb.(string))
		}
	}

	if response == "" {
		return "Unable to find torrents for your search"
	}

	return response
}

func torrentResponse(torrents structures.Torrents) string {
	var response string

	for i, torrent := range torrents {
		seeders, err2 := strconv.Atoi(torrent.Seeders)
		if err2 != nil {
			Logger.Printf("ERR torrent seeder Atoi: %s\n", err2.Error())
			continue
		}
		size, err3 := strconv.ParseUint(torrent.Size, 10, 64)
		if err3 != nil {
			Logger.Printf("ERR torrent size Atoi: %s\n", err3.Error())
			continue
		}

		if seeders > 10 {
			response += prepareLink(
				i, torrent.InfoHash, torrent.Name,
				seeders, calculateHumanSize(size), torrent.Imdb)
		}
	}

	if response == "" {
		return "Unable to find torrents for your search"
	}
	return response
}

func calculateHumanSize(size uint64) string {
	humanSize := float64(size / (1024 * 1024))
	sizeSuffix := fmt.Sprintf("*%.0f MiB*", humanSize)
	if humanSize > 999 {
		humanSize = humanSize / 1024
		sizeSuffix = fmt.Sprintf("*%.1f GiB*", humanSize)
	}
	return sizeSuffix
}

func prepareLink(i int, magnetLink string, torrentName string,
	torrentSeeders int, sizeSuffix string, imdb string) string {
	var response string

	magnetLink = fmt.Sprintf("magnet/?xt=urn:btih:%s", magnetLink)
	response += fmt.Sprintf(
		"%d: <http://%s|%s> Seeds:%d %s", i, magnetLink,
		torrentName, torrentSeeders, sizeSuffix)

	if imdb != "" {
		response += fmt.Sprintf(" (<https://www.imdb.com/title/%s|imdb>)", imdb)
	}

	return response + "\n"
}

func getTorrentsFromJSON(jsonObject []byte) *structures.Torrents {
	var s = new(structures.Torrents)
	err := json.Unmarshal(jsonObject, &s)
	if err != nil {
		Logger.Printf("ERR: %s => %s", err, jsonObject)
	}

	return s
}

func getTop100FromJSON(jsonObject []byte) *structures.Top100Movies {
	var s = new(structures.Top100Movies)
	err := json.Unmarshal(jsonObject, &s)
	if err != nil {
		Logger.Printf("ERR: %s => %s", err, jsonObject)
	}

	return s
}
