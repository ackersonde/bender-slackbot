package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/ackersonde/bender-slackbot/structures"
)

var ethAddrMetaMask = os.Getenv("CTX_ETHEREUM_ADDRESS_METAMASK")
var etherscanAPIKey = os.Getenv("CTX_ETHERSCAN_API_KEY")
var stellarAccount = os.Getenv("CTX_STELLAR_LUMENS_ADDRESS")
var pgpKey = os.Getenv("CTX_CURRENT_PGP_FINGERPRINT")

func checkStellarLumensValue() string {
	response := ""

	accountBalanceStr := getStellarLumens()
	if !strings.HasPrefix(accountBalanceStr, "ERR") {
		stellarPriceUSDStr := getStellarPrice()
		if !strings.HasPrefix(stellarPriceUSDStr, "ERR") {
			accountBalance, err := strconv.ParseFloat(accountBalanceStr, 64)
			if err == nil {
				stellarPriceUSD, err := strconv.ParseFloat(stellarPriceUSDStr, 64)
				if err == nil {
					response = fmt.Sprintf(
						"Your %f :lumens: is worth $%f (<https://stellar.expert/explorer/public/account/%s|stellar.expert>)",
						accountBalance, stellarPriceUSD*accountBalance, stellarAccount)
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

func getStellarPrice() string {
	response := ""

	stellarPriceResp, err := http.Get("https://api.stellarterm.com/v1/ticker.json")
	if err != nil {
		response = fmt.Sprintf("ERR: stellar Lumens http.Get: %s", err)
	} else {
		defer stellarPriceResp.Body.Close()
		stellarPriceJSON, _ := ioutil.ReadAll(stellarPriceResp.Body)
		// "externalPrices":{"USD_BTC":9650.16,"BTC_XLM":0.00000717,"USD_XLM":0.069192,"USD_XLM_24hAgo":0.070537,"USD_XLM_change":-1.90748}

		re := regexp.MustCompile(`"USD_XLM":(?P<price>[0-9]+\.?[0-9]*),`)
		matches := re.FindAllStringSubmatch(string(stellarPriceJSON), -1)
		names := re.SubexpNames()

		m := map[string]string{}
		for i, n := range matches[0] {
			m[names[i]] = n
		}
		Logger.Println(" @ $" + m["price"])
		response = m["price"]
	}

	return response
}

func getStellarLumens() string {
	response := ""

	stellarLumensURL := fmt.Sprintf("https://horizon.stellar.org/accounts/%s", stellarAccount)
	// {"status":"1","message":"OK","result":{"ethbtc":"0.0217","ethbtc_timestamp":"1589119180","ethusd":"190.57","ethusd_timestamp":"1589119172"}}
	stellarResp, err := http.Get(stellarLumensURL)
	if err != nil {
		response = fmt.Sprintf("ERR: stellar Lumens http.Get: %s", err)
	} else {
		defer stellarResp.Body.Close()
		stellarLumensLedger := new(structures.StellarLumensLedger)
		stellarLumensJSON, err2 := ioutil.ReadAll(stellarResp.Body)

		if err2 == nil {
			json.Unmarshal([]byte(stellarLumensJSON), &stellarLumensLedger)
			// find correct Balance -> AssetType == "native"
			for _, balance := range stellarLumensLedger.Balances {
				if balance.AssetType == "native" {
					response = balance.Balance
					break
				}
			}
		} else {
			response = fmt.Sprintf("ERR: stellar Lumens ioutil.ReadAll: %s", err2)
		}
	}

	return response
}

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
					response = fmt.Sprintf(
						"Your %f :ethereum: is worth $%f (<https://etherscan.io/address/%s|etherscan.io>)",
						accountBalance, ethereumPriceUSD*accountBalance, ethAddrMetaMask)
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

	// https://min-api.cryptocompare.com/data/price?fsym=ETH&tsyms=EUR => {"EUR":187.32}

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
	credentials := fmt.Sprintf("&address=0x%s&apikey=%s",
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

func pgpKeys() string {
	currentPGPKey := "<https://keyserver.ubuntu.com/pks/lookup?search=0x" +
		pgpKey + "&fingerprint=on&op=index|" + string(pgpKey[len(pgpKey)-16:]) + ">"
	pastPGPKeys := "<https://keyserver.ubuntu.com/pks/lookup?search=Dan+Ackerson&fingerprint=on&op=index|'Dan Ackerson'>"
	keybaseIdentify := "<https://keybase.io/danackerson|danackerson>"
	keybasePGP := "<https://keybase.io/danackerson/pgp_keys.asc?fingerprint=" +
		pgpKey + "|" + string(pgpKey[len(pgpKey)-16:]) + ">"

	// TODO: check actual current PGP key value from keyserver.ubuntu.com against keybase PGP key value!

	return fmt.Sprintf(":ubuntu: %s & %s\n:keybase: %s & %s",
		currentPGPKey, pastPGPKeys, keybasePGP, keybaseIdentify)
}
