package structures

import "time"

type PhotoPrismLinks []struct {
	UID         string    `json:"UID"`
	Share       string    `json:"Share"`
	Slug        string    `json:"Slug"`
	Token       string    `json:"Token"`
	Expires     int       `json:"Expires"`
	Views       int       `json:"Views"`
	MaxViews    int       `json:"MaxViews"`
	HasPassword bool      `json:"HasPassword"`
	CanComment  bool      `json:"CanComment"`
	CanEdit     bool      `json:"CanEdit"`
	CreatedAt   time.Time `json:"CreatedAt"`
	ModifiedAt  time.Time `json:"ModifiedAt"`
}

type PhotoPrismAlbums []struct {
	UID            string    `json:"UID"`
	ParentUID      string    `json:"ParentUID"`
	Thumb          string    `json:"Thumb"`
	Slug           string    `json:"Slug"`
	Type           string    `json:"Type"`
	Title          string    `json:"Title"`
	Location       string    `json:"Location"`
	Category       string    `json:"Category"`
	Caption        string    `json:"Caption"`
	Description    string    `json:"Description"`
	Notes          string    `json:"Notes"`
	Filter         string    `json:"Filter"`
	Order          string    `json:"Order"`
	Template       string    `json:"Template"`
	Path           string    `json:"Path"`
	State          string    `json:"State"`
	Country        string    `json:"Country"`
	Year           int       `json:"Year"`
	Month          int       `json:"Month"`
	Day            int       `json:"Day"`
	Favorite       bool      `json:"Favorite"`
	Private        bool      `json:"Private"`
	PhotoCount     int       `json:"PhotoCount"`
	LinkCount      int       `json:"LinkCount"`
	CreatedAt      time.Time `json:"CreatedAt"`
	UpdatedAt      time.Time `json:"UpdatedAt"`
	DeletedAt      time.Time `json:"DeletedAt"`
	PublicURL      string
	ExpiringInDays int
	Views          int
}
