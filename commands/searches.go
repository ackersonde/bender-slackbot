package commands

// forked from https://github.com/jasonrhansen/piratebay
import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
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

// https://pirateproxy.wtf/
var proxies = []string{"tpb.cool", "piratebay.tech", "thepiratebay.fail", "piratebay.icu", "thepirate.host"}

func searchProxy(url string) *html.Node {
	for _, proxy := range proxies {
		uri := "https://" + proxy + url
		req, err := http.NewRequest("GET", uri, nil)
		if err != nil {
			log.Printf("http.NewRequest() failed with '%s'\n", err)
			continue
		}

		// create a context indicating 100 ms timeout
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*5000)
		defer cancel()
		// get a new request based on original request but with the context
		req = req.WithContext(ctx)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			// the request should timeout because we want to wait max 100 ms
			// but the server doesn't return response for 3 seconds
			log.Printf("http.DefaultClient.Do() failed with:\n'%s'\n", err)
			continue
		}
		if resp != nil {
			doc, err := html.Parse(resp.Body)
			defer resp.Body.Close()

			if err == nil && doc != nil {
				log.Printf("%s%s succeeded!", proxy, url)
			}

			return doc
		}
	}

	return nil
}

// Torrent stores information about a torrent found on The Pirate Bay.
type Torrent struct {
	Title       string
	MagnetLink  string
	Size        int
	Uploaded    string
	User        string
	VIP         bool
	Trusted     bool
	Seeders     int
	Leechers    int
	Category    string
	CategoryID  int
	DetailsLink string
}

// SearchFor is now commented
func SearchFor(term string, cat Category) ([]Torrent, string) {
	response := ""

	var torrents []Torrent
	torrents, err := search(term, cat)
	found := 0
	if err == nil {
		for i, t := range torrents {
			if t.Seeders > 10 {
				found++
				humanSize := float64(t.Size / (1024 * 1024))
				sizeSuffix := fmt.Sprintf("*%.0f MiB*", humanSize)
				if humanSize > 999 {
					humanSize = humanSize / 1024
					sizeSuffix = fmt.Sprintf("*%.1f GiB*", humanSize)
				}
				response += fmt.Sprintf("%d: <http://%s|%s> Seeds:%d %s\n", i, t.MagnetLink, t.Title, t.Seeders, sizeSuffix)
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
func search(query string, cats ...Category) ([]Torrent, error) {
	var doc *html.Node

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

		searchStringURL := "/search/" + url.QueryEscape(query) + "/0/99/" + catStr
		doc = searchProxy(searchStringURL)
	} else {
		doc = searchProxy("/browse/207/0/7/0")
	}

	if doc == nil {
		return nil, errors.New("unable to contact any PB Proxy...try again later")
	}

	return getTorrentsFromDoc(doc), nil
}

func getTorrentsFromDoc(doc *html.Node) []Torrent {
	tc := make(chan Torrent)
	go func() {
		loopDOM(doc, tc)
		close(tc)
	}()
	var torrents []Torrent
	for t := range tc {
		torrents = append(torrents, t)
	}

	return torrents
}

func loopDOM(n *html.Node, tc chan Torrent) {
	if n.Type == html.ElementNode && n.Data == "tbody" {
		getTorrentsFromTable(n, tc)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		loopDOM(c, tc)
	}
}

func getTorrentsFromTable(n *html.Node, tc chan Torrent) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "tr" {
			getTorrentFromRow(c, tc)
		}
	}
}

func getTorrentFromRow(n *html.Node, tc chan Torrent) {
	var torrent Torrent
	col := 0
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "td" {
			setTorrentDataFromCell(c, &torrent, col)
			col++
		}
	}
	tc <- torrent
}

func setTorrentDataFromCell(n *html.Node, t *Torrent, col int) {
	if n.Type == html.ElementNode {
		if col == 2 {
			if s, err := strconv.Atoi(getNodeText(n)); err == nil {
				t.Seeders = s
			}
		} else if col == 3 {
			if l, err := strconv.Atoi(getNodeText(n)); err == nil {
				t.Leechers = l
			}
		} else {
			for _, a := range n.Attr {
				if n.Data == "a" && a.Key == "href" {
					if strings.HasPrefix(a.Val, "magnet") {
						t.MagnetLink = a.Val
					} else if strings.Contains(a.Val, "/torrent/") {
						if t.Title == "" {
							t.Title = getNodeText(n)
							t.DetailsLink = a.Val
						}
					}
				} else if n.Data == "font" && a.Key == "class" && a.Val == "detDesc" {
					parts := strings.Split(getNodeText(n), ", ")
					if len(parts) > 1 {
						t.Uploaded = strings.Split(parts[0], " ")[1]
						t.Size = sizeStrToInt(strings.Split(parts[1], " ")[1])
					}
				}
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		setTorrentDataFromCell(c, t, col)
	}
}

func sizeStrToInt(s string) int {
	var multiply int
	if len(s) < 5 {
		return 0
	}
	multiply = 1
	ext := s[len(s)-3:]
	if ext == "MiB" {
		multiply = 1024 * 1024
	} else if ext == "KiB" {
		multiply = 1024
	} else if ext == "GiB" {
		multiply = 1024 * 1024 * 1024
	}
	size, err := strconv.ParseFloat(s[:len(s)-5], 64)
	if err != nil {
		return 0
	}
	return int(size * float64(multiply))
}

func getNodeText(n *html.Node) string {
	for a := n.FirstChild; a != nil; a = a.NextSibling {
		if a.Type == html.TextNode {
			return strings.TrimSpace(a.Data)
		}
	}
	return ""
}
