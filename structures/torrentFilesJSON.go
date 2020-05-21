package structures

// Torrents response object from pb
type Torrents []struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	InfoHash string `json:"info_hash"`
	Leechers string `json:"leechers"`
	Seeders  string `json:"seeders"`
	NumFiles string `json:"num_files"`
	Size     string `json:"size"`
	Username string `json:"username"`
	Added    string `json:"added"`
	Status   string `json:"status"`
	Category string `json:"category"`
	Imdb     string `json:"imdb"`
}

// Top100Movies listing has an Anon field
type Top100Movies []struct {
	ID       int         `json:"id"`
	InfoHash string      `json:"info_hash"`
	Category int         `json:"category"`
	Name     string      `json:"name"`
	Status   string      `json:"status"`
	NumFiles int         `json:"num_files"`
	Size     uint64      `json:"size"`
	Seeders  int         `json:"seeders"`
	Leechers int         `json:"leechers"`
	Username string      `json:"username"`
	Added    int         `json:"added"`
	Anon     int         `json:"anon,omitempty"`
	Imdb     interface{} `json:"imdb"` // e.g. https://www.imdb.com/title/<Imdb-string>
}
