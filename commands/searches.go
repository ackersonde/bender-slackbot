package commands

// forked from https://github.com/jasonrhansen/piratebay
import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
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

const pbProxiesURL = "https://piratebay-proxylist.se/api/v1/proxies"
const proxyFile = "/tmp/pbproxies.json"

var proxyIndex = 0

// InitPBProxies json file on disk
func InitPBProxies() time.Time {
	resp, _ := http.Get(pbProxiesURL)
	if resp.StatusCode != 200 {
		log.Printf("ERR: Unable to scan %s (%d)", pbProxiesURL, resp.StatusCode)
	} else {
		defer resp.Body.Close()
		htmlData, _ := ioutil.ReadAll(resp.Body) //<--- here!
		if len(htmlData) > 0 {
			err := ioutil.WriteFile(proxyFile, htmlData, 0644)
			if err != nil {
				log.Printf("ERR: Unable to save %s: %s", proxyFile, err.Error())
			}
		}
	}

	fileInfo, err := os.Stat(proxyFile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("ERR: %s does not exist.", proxyFile)
		}
	} else if fileInfo.Size() > 0 {
		log.Printf("INFO: %s from %s", proxyFile, fileInfo.ModTime().Format("Jan _2, 2006 @15:04"))
	}

	return fileInfo.ModTime()
}

func loadPBProxies() []interface{} {
	var pbProxiesJSON map[string]interface{}

	data, err := ioutil.ReadFile(proxyFile)
	if err != nil {
		log.Printf("ERR: %s", err)
		return nil
	}

	json.Unmarshal([]byte(data), &pbProxiesJSON)
	return pbProxiesJSON["proxies"].([]interface{})
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

	fmt.Printf("searching for: '%s' in category %v\n", term, cat)

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
				response += fmt.Sprintf("%d: <https://%s|%s> Seeds:%d %s\n", i, t.MagnetLink, t.Title, t.Seeders, sizeSuffix)
			}
		}
	} else {
		response = pbProxiesURL + " seems to be offline: " + fmt.Sprintf("%v", err)
	}

	if found < 1 {
		response = "Unable to find torrents with enough Seeders for '" + term + "'"
	}

	return torrents, response
}

func findWorkingProxy() (*http.Response, error) {
	resp := new(http.Response)

	var domain map[string]interface{}
	var proxyURL string
	var lastModTime time.Time

	proxies := loadPBProxies()
	if len(proxies) == 0 {
		lastModTime = InitPBProxies()
		proxies = loadPBProxies()
	}

	if len(proxies) == 0 {
		errString := "ERR: " + pbProxiesURL + " offline. " +
			"Unable to get any proxies as of " +
			lastModTime.Format("Jan _2, 2016 @15:04")
		return resp, errors.New(errString)
	}

	err := errors.New("")
	for i := 0; resp == nil || resp.StatusCode != 200; i++ {
		resp, err = http.Get(proxyURL + "/browse/207/0/7/0")
		if err != nil || resp.StatusCode != 200 {
			domain = proxies[0].(map[string]interface{})
			proxyURL = "https://" + domain["domain"].(string)
		}
	}

	return resp, err
}

// search returns the torrents found with the given search string and categories.
func search(query string, cats ...Category) ([]Torrent, error) {
	var err error

	resp, err := findWorkingProxy()
	proxyURL := resp.Request.Host

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

		searchStringURL := "https://" + proxyURL + "/search/" + url.QueryEscape(query) + "/0/99/" + catStr
		log.Printf("searching for: %s", searchStringURL)
		resp, err = http.Get(searchStringURL)
		if err != nil {
			return nil, err
		}
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
	}

	return getTorrentsFromDoc(doc, proxyURL), nil
}

func getTorrentsFromDoc(doc *html.Node, domain string) []Torrent {
	tc := make(chan Torrent)
	go func() {
		loopDOM(doc, tc, domain)
		close(tc)
	}()
	var torrents []Torrent
	for t := range tc {
		torrents = append(torrents, t)
	}

	return torrents
}

func loopDOM(n *html.Node, tc chan Torrent, domain string) {
	if n.Type == html.ElementNode && n.Data == "tbody" {
		getTorrentsFromTable(n, tc, domain)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		loopDOM(c, tc, domain)
	}
}

func getTorrentsFromTable(n *html.Node, tc chan Torrent, domain string) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "tr" {
			getTorrentFromRow(c, tc, domain)
		}
	}
}

func getTorrentFromRow(n *html.Node, tc chan Torrent, domain string) {
	var torrent Torrent
	col := 0
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "td" {
			setTorrentDataFromCell(c, &torrent, col, domain)
			col++
		}
	}
	tc <- torrent
}

func setTorrentDataFromCell(n *html.Node, t *Torrent, col int, domain string) {
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
					var b bytes.Buffer
					log.Printf("Node: %s", html.Render(&b, n))

					if strings.HasPrefix(a.Val, "magnet") {
						t.MagnetLink = a.Val
					} else if strings.HasPrefix(a.Val, "/torrent/") {
						if t.Title == "" {
							t.Title = getNodeText(n)
							t.DetailsLink = domain + a.Val
						}
					} else if strings.HasPrefix(a.Val, "/browse/") && t.Category == "" {
						t.Category = getNodeText(n)
					}
				} else if n.Data == "font" && a.Key == "class" && a.Val == "detDesc" {
					parts := strings.Split(getNodeText(n), ", ")
					if len(parts) > 1 {
						t.Uploaded = strings.Split(parts[0], " ")[1]
						t.Size = sizeStrToInt(strings.Split(parts[1], " ")[1])
					}
				} else if n.Data == "img" && a.Key == "alt" && a.Val == "VIP" {
					t.VIP = true
				} else if n.Data == "img" && a.Key == "alt" && a.Val == "Trusted" {
					t.Trusted = true
				} else if n.Data == "a" && a.Key == "class" && a.Val == "detDesc" {
					t.User = getNodeText(n)
				}
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		setTorrentDataFromCell(c, t, col, domain)
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
