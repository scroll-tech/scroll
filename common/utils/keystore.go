package utils

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/scroll-tech/go-ethereum/accounts/keystore"
	"github.com/scroll-tech/go-ethereum/log"
)

// LoadOrCreateKey load or create keystore by keystorePath,  keystorePath cannot be a dir.
func LoadOrCreateKey(keystorePath string, keystorePassword string) (*ecdsa.PrivateKey, error) {
	if fi, err := os.Stat(keystorePath); os.IsNotExist(err) {
		// If there is no keystore, make a new one.
		ks := keystore.NewKeyStore(filepath.Dir(keystorePath), keystore.StandardScryptN, keystore.StandardScryptP)
		account, kerr := ks.NewAccount(keystorePassword)
		if kerr != nil {
			return nil, fmt.Errorf("generate crypto account failed %v", kerr)
		}

		err = os.Rename(account.URL.Path, keystorePath)
		if err != nil {
			return nil, err
		}
		log.Info("create a new account", "address", account.Address.Hex())
	} else if err != nil {
		return nil, err
	} else if fi.IsDir() {
		return nil, errors.New("keystorePath cannot be a dir")
	}

	keyjson, err := os.ReadFile(filepath.Clean(keystorePath))
	if err != nil {
		return nil, err
	}

	key, err := keystore.DecryptKey(keyjson, keystorePassword)
	if err != nil {
		return nil, err
	}
	return key.PrivateKey, nil
}
