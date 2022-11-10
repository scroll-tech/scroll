package utils

import (
	"crypto/ecdsa"
	"fmt"
	"os"
	"path/filepath"

	"github.com/scroll-tech/go-ethereum/accounts/keystore"
	"github.com/scroll-tech/go-ethereum/log"
)

func LoadOrCreateKey(keystorePath string, keystorePassword string) (*ecdsa.PrivateKey, error) {
	if fi, err := os.Stat(keystorePath); os.IsNotExist(err) {
		// If there is no keystore, make a new one.
		ks := keystore.NewKeyStore(filepath.Dir(keystorePath), keystore.StandardScryptN, keystore.StandardScryptP)
		account, err := ks.NewAccount(keystorePassword)
		if err != nil {
			return nil, fmt.Errorf("generate crypto account failed %v", err)
		}

		err = os.Rename(account.URL.Path, keystorePath)
		if err != nil {
			return nil, err
		}
		log.Info("create a new account", "address", account.Address.Hex())
	} else if err != nil {
		return nil, err
	} else if fi.IsDir() {
		return nil, fmt.Errorf("keystorePath cannot be a dir")
	}

	keyjson, err := os.ReadFile(keystorePath)
	if err != nil {
		return nil, err
	}

	key, err := keystore.DecryptKey(keyjson, keystorePassword)
	if err != nil {
		return nil, err
	}
	return key.PrivateKey, nil
}
