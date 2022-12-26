package config

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
)

// UnmarshalMinBalance : unmarshal min balance.
func UnmarshalMinBalance(input string) (*big.Int, error) {
	minBalance, failed := new(big.Int).SetString(input, 10)
	if failed {
		minBalance, _ = new(big.Int).SetString("100000000000000000000", 10)
		return minBalance, nil
	}
	return minBalance, nil
}

// UnmarshalPrivateKeys : unmarshal private keys.
func UnmarshalPrivateKeys(input []string) ([]*ecdsa.PrivateKey, error) {
	// Get messenger private key list.
	var privateKeys []*ecdsa.PrivateKey
	for _, privStr := range input {
		priv, err := crypto.ToECDSA(common.FromHex(privStr))
		if err != nil {
			return nil, fmt.Errorf("incorrect private_key_list format, err: %v", err)
		}
		privateKeys = append(privateKeys, priv)
	}
	return privateKeys, nil
}
