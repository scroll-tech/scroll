package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

type BinanceResponse struct {
	Price string `json:"price"`
}

func GetExchangeRateFromBinanceApi(endpoint string) (float64, error) {
	// make HTTP GET request
	resp, err := http.Get(endpoint)
	if err != nil {
		return 0, fmt.Errorf("error making HTTP request: %w", err)
	}
	defer resp.Body.Close()

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

	if err := resp.Body.Close(); err != nil {
		return 0, fmt.Errorf("error closing response body: %v", err)
	}

	return price, nil
}
