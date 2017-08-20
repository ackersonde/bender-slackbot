package commands

// forked from https://github.com/jasonrhansen/piratebay
import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

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

const pirateURL = "https://thepiratebay.org"

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
				response += fmt.Sprintf("%d: <http://%s|%s> Seeds:%d %s\n", i, t.MagnetLink, t.Title, t.Seeders, sizeSuffix)
			}
		}
	} else {
		response = pirateURL + " seems to be offline: " + fmt.Sprintf("%v", err)
	}

	if found < 1 {
		response = "Unable to find torrents with enough Seeders for '" + term + "'"
	}

	return torrents, response
}

// search returns the torrents found with the given search string and categories.
func search(query string, cats ...Category) ([]Torrent, error) {
	resp := new(http.Response)
	var err error

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

		searchStringURL := pirateURL + "/search/" + url.QueryEscape(query) + "/0/99/" + catStr
		fmt.Println("searching for: " + searchStringURL)
		resp, err = http.Get(searchStringURL)
		if err != nil {
			return nil, err
		}
	} else {
		resp, _ = http.Get(pirateURL + "/browse/207/0/7/0")
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
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
					} else if strings.HasPrefix(a.Val, "/torrent/") {
						if t.Title == "" {
							t.Title = getNodeText(n)
							t.DetailsLink = pirateURL + a.Val
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
