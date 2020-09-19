package structures

// Torrents response object from pb
// https://tpb.cool/api?url=/q.php?q=Walking+Dead
type Torrents []struct {
	Added    string `json:"added"`
	Category string `json:"category"`
	ID       string `json:"id"`
	Imdb     string `json:"imdb"`
	InfoHash string `json:"info_hash"`
	Leechers string `json:"leechers"`
	Name     string `json:"name"`
	NumFiles string `json:"num_files"`
	Seeders  string `json:"seeders"`
	Size     string `json:"size"`
	Status   string `json:"status"`
	Username string `json:"username"`
}

// Top100Movies listing has an Anon field
// https://tpb.cool/api?url=/precompiled/data_top100_207.json
type Top100Movies []struct {
	Added    int         `json:"added"`
	Anon     int         `json:"anon,omitempty"`
	Category int         `json:"category"`
	ID       int         `json:"id"`
	Imdb     interface{} `json:"imdb"` // e.g. https://www.imdb.com/title/<Imdb-string>
	InfoHash string      `json:"info_hash"`
	Leechers int         `json:"leechers"`
	Name     string      `json:"name"`
	NumFiles int         `json:"num_files"`
	Seeders  int         `json:"seeders"`
	Size     uint64      `json:"size"`
	Status   string      `json:"status"`
	Username string      `json:"username"`
}
