package utils

import "testing"

func TestGetExchangeRateFromBinanceApi(t *testing.T) {
	endpoint := "https://api.binance.com/api/v3/ticker/price?symbol=UNIETH"
	_, err := GetExchangeRateFromBinanceApi(endpoint)
	if err != nil {
		t.Fatalf("Error getting exchange rate from Binance API: %v", err)
	}
}