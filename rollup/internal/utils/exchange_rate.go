package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/scroll-tech/go-ethereum/log"
)

var BinanceApiEndpoint string = "https://api.binance.com/api/v3/ticker/price?symbol=%s"

type BinanceResponse struct {
	Price string `json:"price"`
}

func GetExchangeRateFromBinanceApi(tokenSymbolPair string) (float64, error) {
	// make HTTP GET request
	resp, err := http.Get(fmt.Sprintf(BinanceApiEndpoint, tokenSymbolPair))
	if err != nil {
		return 0, fmt.Errorf("error making HTTP request: %w", err)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Error("error closing response body", "err", err)
		}
	}()

	// check for successful response
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("error reading response body: %w", err)
	}

	// unmarshal JSON response
	var data BinanceResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return 0, fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	// convert price string to float64
	price, err := strconv.ParseFloat(data.Price, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing price string: %w", err)
	}

	return price, nil
}
