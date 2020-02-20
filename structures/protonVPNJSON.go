package structures

// ProtonVPNServers is a representation of the Proton VPN servers
// found at https://api.protonmail.ch/vpn/logicals
type ProtonVPNServers struct {
	Code           int `json:"Code"`
	LogicalServers []LogicalServer `json:"LogicalServers"`
}

// LogicalServer is what we use to update VPN server details
type LogicalServer struct {
	Name         string      `json:"Name"`
	EntryCountry string      `json:"EntryCountry"`
	ExitCountry  string      `json:"ExitCountry"`
	Domain       string      `json:"Domain"`
	Tier         int         `json:"Tier"`
	Features     int         `json:"Features"`
	Region       interface{} `json:"Region"`
	City         string      `json:"City"`
	ID           string      `json:"ID"`
	Location     struct {
		Lat  float64 `json:"Lat"`
		Long float64 `json:"Long"`
	} `json:"Location"`
	Status  int `json:"Status"`
	Servers []struct {
		EntryIP string `json:"EntryIP"`
		ExitIP  string `json:"ExitIP"`
		Domain  string `json:"Domain"`
		ID      string `json:"ID"`
		Status  int    `json:"Status"`
	} `json:"Servers"`
	Load  int     `json:"Load"`
	Score float64 `json:"Score"`
}