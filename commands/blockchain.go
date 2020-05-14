package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var ethAddrMetaMask = os.Getenv("CTX_ETHEREUM_ADDRESS_METAMASK")
var etherscanAPIKey = os.Getenv("CTX_ETHERSCAN_API_KEY")

func checkEthereumValue() string {
	response := ""

	accountBalanceStr := getEthereumTokens() // + " ETH :ethereum:"
	if !strings.HasPrefix(accountBalanceStr, "ERR") {
		ethereumPriceUSDStr := getEthereumPrice()
		if !strings.HasPrefix(ethereumPriceUSDStr, "ERR") {
			accountBalance, err := strconv.ParseFloat(accountBalanceStr, 64)
			if err == nil {
				ethereumPriceUSD, err := strconv.ParseFloat(ethereumPriceUSDStr, 64)
				if err == nil {
					response = fmt.Sprintf("Your %f :ethereum: is worth $%f",
						accountBalance, ethereumPriceUSD*accountBalance)
				} else {
					response = err.Error()
				}
			} else {
				response = err.Error()
			}
		}
	}

	return response
}

func getEthereumPrice() string {
	response := ""

	ethereumPriceURL := fmt.Sprintf("https://api.etherscan.io/api?module=stats&action=ethprice&apikey=%s", etherscanAPIKey)
	// {"status":"1","message":"OK","result":{"ethbtc":"0.0217","ethbtc_timestamp":"1589119180","ethusd":"190.57","ethusd_timestamp":"1589119172"}}
	Logger.Println(ethereumPriceURL)
	ethereumResp, err := http.Get(ethereumPriceURL)
	if err != nil {
		response = fmt.Sprintf("ERR: etherscan EthPrice http.Get: %s", err)
	} else {
		defer ethereumResp.Body.Close()
		ethereumPriceJSON, err2 := ioutil.ReadAll(ethereumResp.Body)

		if err2 == nil {
			var result map[string]map[string]string
			json.Unmarshal([]byte(ethereumPriceJSON), &result)
			Logger.Println(result)

			ethereumPrice, err3 := strconv.ParseFloat(string(result["result"]["ethusd"]), 64)
			if err3 == nil {
				response = fmt.Sprintf("%f", ethereumPrice)
			} else {
				response = fmt.Sprintf("ERR: etherscan EthPrice ParseFloat: %s", err3)
			}
		} else {
			response = fmt.Sprintf("ERR: etherscan EthPrice ioutil.ReadAll: %s", err2)
		}
	}

	return response
}

func getEthereumTokens() string {
	response := ""
	credentials := fmt.Sprintf("&address=%s&apikey=%s",
		ethAddrMetaMask,
		etherscanAPIKey)

	etherscanAccountBalanceURL := fmt.Sprintf(
		"https://api.etherscan.io/api?module=account&tag=latest&action=balance%s",
		credentials)
	Logger.Println(etherscanAccountBalanceURL)

	accountBalanceWeiResp, err := http.Get(etherscanAccountBalanceURL)
	if err != nil {
		response = fmt.Sprintf("ERR: etherscan AcctBal http.Get: %s", err)
	} else {
		defer accountBalanceWeiResp.Body.Close()
		accountBalanceWeiJSON, err2 := ioutil.ReadAll(accountBalanceWeiResp.Body)

		if err2 == nil {
			var result map[string]string
			json.Unmarshal([]byte(accountBalanceWeiJSON), &result)
			Logger.Println(result)

			accountBalanceWei, err3 := strconv.ParseFloat(result["result"], 64)
			if err3 == nil {
				accountBalance := accountBalanceWei / 1000000000000000000
				response = fmt.Sprintf("%f", accountBalance)
			} else {
				response = fmt.Sprintf("ERR: etherscan AcctBal ParseFloat: %s", err3)
			}
		} else {
			response = fmt.Sprintf("ERR: etherscan AcctBal ioutil.ReadAll: %s", err2)
		}
	}

	return response
}
