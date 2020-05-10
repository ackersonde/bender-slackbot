package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
)

func checkEthereumValue() string {
	response := ""
	ethAddrMetaMask := os.Getenv("CTX_ETHEREUM_ADDRESS_METAMASK")
	etherscanAPIKey := os.Getenv("CTX_ETHERSCAN_API_KEY")
	credentials := fmt.Sprintf("&address=%s&tag=latest&apikey=%s", ethAddrMetaMask, etherscanAPIKey)

	etherscanAccountBalanceURL := fmt.Sprintf("https://api.etherscan.io/api?module=account&action=balance%s", credentials)
	accountBalanceWeiResp, err := http.Get(etherscanAccountBalanceURL)
	if err != nil {
		response = "etherscan AcctBal http.Get ERR: %s" + err.Error()
	} else {
		defer accountBalanceWeiResp.Body.Close()
		accountBalanceWeiJSON, err2 := ioutil.ReadAll(accountBalanceWeiResp.Body)

		if err2 != nil {
			var result map[string]string
			json.Unmarshal([]byte(accountBalanceWeiJSON), &result)
			fmt.Println(result)
			for key, value := range result {
				fmt.Println(key, value)
			}
			accountBalanceWeiStr := result["result"]
			accountBalanceWei, err3 := strconv.ParseFloat(accountBalanceWeiStr, 64)
			if err3 == nil {
				accountBalance := accountBalanceWei / 1000000000000000000
				response = fmt.Sprintf("%f", accountBalance) + " ETH :ethereum:"
			} else {
				response = "etherscan AcctBal ParseFloat ERR: " + err3.Error()
			}
		} else {
			response = "etherscan AcctBal ioutil.ReadAll ERR: " + err2.Error()
		}
	}

	// TODO: now get the price so we can calc our NetWorth
	//ethereumPrice := fmt.Sprintf("https://api.etherscan.io/api?module=stats&action=ethprice&apikey=%s", etherscanAPIKey)
	// {"status":"1","message":"OK","result":{"ethbtc":"0.0217","ethbtc_timestamp":"1589119180","ethusd":"190.57","ethusd_timestamp":"1589119172"}}

	// result.ethusd * accountBalance / 1000000000000000000 = Value

	return response
}
