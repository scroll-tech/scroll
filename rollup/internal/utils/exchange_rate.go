package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/scroll-tech/go-ethereum/log"
)

var BinanceApiEndpoint string = "https://api.binance.com/api/v3/ticker/price?symbol=%s"

type BinanceResponse struct {
	Price string `json:"price"`
}

func GetExchangeRateFromBinanceApi(tokenSymbolPair string, maxRetries int) (float64, error) {
	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			time.Sleep(5 * time.Second)
		}

		// make HTTP GET request
		resp, err := http.Get(fmt.Sprintf(BinanceApiEndpoint, tokenSymbolPair))
		if err != nil {
			log.Error("error making HTTP request", "err", err)
			continue
		}
		defer func() {
			err = resp.Body.Close()
			if err != nil {
				log.Error("error closing response body", "err", err)
			}
		}()

		// check for successful response
		if resp.StatusCode != http.StatusOK {
			log.Error("unexpected status code", "code", resp.StatusCode)
			continue
		}

		// read response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Error("error reading response body", "err", err)
			continue
		}

		// unmarshal JSON response
		var data BinanceResponse
		err = json.Unmarshal(body, &data)
		if err != nil {
			log.Error("error unmarshaling JSON", "err", err)
			continue
		}

		// convert price string to float64
		price, err := strconv.ParseFloat(data.Price, 64)
		if err != nil {
			log.Error("error parsing price string", "err", err)
			continue
		}

		// successful response, return price
		return price, nil
	}

	// all retries failed, return error
	return 0, fmt.Errorf("failed to get exchange rate after %d retries", maxRetries)
}
