package viper

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/log"
)

// Get : Get interface type config.
func (v *Viper) Get(key string) interface{} {
	v.mu.RLock()
	defer v.mu.RUnlock()
	//fmt.Println("Get key: ", key, " value: ", v.vp.Get(key))
	return v.vp.Get(key)
}

// GetBool : Get bool type config.
func (v *Viper) GetBool(key string) bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	//fmt.Println("GetBool key: ", key, " value: ", v.vp.GetBool(key))
	return v.vp.GetBool(key)
}

// GetFloat64 : Get float64 type config.
func (v *Viper) GetFloat64(key string) float64 {
	v.mu.RLock()
	defer v.mu.RUnlock()
	//fmt.Println("GetFloat64 key: ", key, " value: ", v.vp.GetFloat64(key))
	return v.vp.GetFloat64(key)
}

// GetInt : Get int type config.
func (v *Viper) GetInt(key string) int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	//fmt.Println("GetInt key: ", key, " value: ", v.vp.GetInt(key))
	return v.vp.GetInt(key)
}

// GetInt64 : Get int64 type config.
func (v *Viper) GetInt64(key string) int64 {
	v.mu.RLock()
	defer v.mu.RUnlock()
	//fmt.Println("GetInt key: ", key, " value: ", v.vp.GetInt64(key))
	return v.vp.GetInt64(key)
}

// GetIntSlice : Get int slice type config.
func (v *Viper) GetIntSlice(key string) []int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	//fmt.Println("GetIntSlice key: ", key, " value: ", v.vp.GetIntSlice(key))
	return v.vp.GetIntSlice(key)
}

// GetString : Get string type config.
func (v *Viper) GetString(key string) string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	//fmt.Println("GetString key: ", key, " value: ", v.vp.GetString(key))
	return v.vp.GetString(key)
}

// GetStringSlice : Get string slice type config.
func (v *Viper) GetStringSlice(key string) []string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	//fmt.Println("GetStringSlice key: ", key, " value: ", v.vp.GetStringSlice(key))
	return v.vp.GetStringSlice(key)
}

// GetAddress : Get address type config.
func (v *Viper) GetAddress(key string) common.Address {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return common.HexToAddress(v.vp.GetString(key))
}

// GetBigInt : Get big.Int type config.
func (v *Viper) GetBigInt(key string) *big.Int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	val := v.vp.GetString(key)
	ret, failed := new(big.Int).SetString(val, 10)
	if !failed {
		ret, _ = new(big.Int).SetString("100000000000000000000", 10)
	}
	return ret
}

// GetECDSAKeys : Get ECDSA keys config.
func (v *Viper) GetECDSAKeys(key string) []*ecdsa.PrivateKey {
	v.mu.RLock()
	defer v.mu.RUnlock()
	keyLists := v.vp.GetStringSlice(key)
	var privateKeys []*ecdsa.PrivateKey
	for _, privStr := range keyLists {
		priv, err := crypto.ToECDSA(common.FromHex(privStr))
		if err != nil {
			log.Error("incorrect private_key_list format", "err", err)
			return nil
		}
		privateKeys = append(privateKeys, priv)
	}
	return privateKeys
}
