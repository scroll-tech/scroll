package utils

import "testing"

func TestGetExchangeRateFromBinanceApi(t *testing.T) {
	tokenSymbolPair := "UNIETH"
	_, err := GetExchangeRateFromBinanceApi(tokenSymbolPair)
	if err != nil {
		t.Fatalf("Error getting exchange rate from Binance API: %v", err)
	}
}