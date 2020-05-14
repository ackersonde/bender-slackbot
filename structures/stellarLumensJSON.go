package structures

// StellarLumensLedger contains all pertinent account information
type StellarLumensLedger struct {
	Links struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
		Transactions struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
		} `json:"transactions"`
		Operations struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
		} `json:"operations"`
		Payments struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
		} `json:"payments"`
		Effects struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
		} `json:"effects"`
		Offers struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
		} `json:"offers"`
		Trades struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
		} `json:"trades"`
		Data struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
		} `json:"data"`
	} `json:"_links"`
	ID                   string `json:"id"`
	AccountID            string `json:"account_id"`
	Sequence             string `json:"sequence"`
	SubentryCount        int    `json:"subentry_count"`
	InflationDestination string `json:"inflation_destination"`
	LastModifiedLedger   int    `json:"last_modified_ledger"`
	Thresholds           struct {
		LowThreshold  int `json:"low_threshold"`
		MedThreshold  int `json:"med_threshold"`
		HighThreshold int `json:"high_threshold"`
	} `json:"thresholds"`
	Flags struct {
		AuthRequired  bool `json:"auth_required"`
		AuthRevocable bool `json:"auth_revocable"`
		AuthImmutable bool `json:"auth_immutable"`
	} `json:"flags"`
	Balances []Balance `json:"balances"`
	Signers  []struct {
		Weight int    `json:"weight"`
		Key    string `json:"key"`
		Type   string `json:"type"`
	} `json:"signers"`
	Data struct {
	} `json:"data"`
	PagingToken string `json:"paging_token"`
}

// Balance contains account information
type Balance struct {
	Balance                           float64 `json:"balance"`
	Limit                             string  `json:"limit,omitempty"`
	BuyingLiabilities                 string  `json:"buying_liabilities"`
	SellingLiabilities                string  `json:"selling_liabilities"`
	LastModifiedLedger                int     `json:"last_modified_ledger,omitempty"`
	IsAuthorized                      bool    `json:"is_authorized,omitempty"`
	IsAuthorizedToMaintainLiabilities bool    `json:"is_authorized_to_maintain_liabilities,omitempty"`
	AssetType                         string  `json:"asset_type"`
	AssetCode                         string  `json:"asset_code,omitempty"`
	AssetIssuer                       string  `json:"asset_issuer,omitempty"`
}
