package viper

import (
	"crypto/ecdsa"
	"math/big"
	"strings"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/spf13/cast"
)

// Get : Get interface type config.
func (v *Viper) Get(key string) interface{} {
	idx := strings.IndexByte(key, '.')
	if idx >= 0 {
		vp, _ := v.data.Load(key[:idx])
		if vip, ok := vp.(*Viper); ok {
			return vip.Get(key[idx+1:])
		}
		return vp
	}
	val, _ := v.data.Load(key)
	return val
}

// Sub : get a viper sub config.
func (v *Viper) Sub(key string) *Viper {
	val := v.Get(key)
	vp, _ := val.(*Viper)
	return vp
}

// Set : set viper config.
func (v *Viper) Set(key string, val interface{}) {
	idx := strings.LastIndexByte(key, '.')
	if idx >= 0 {
		if vp := v.Sub(key[:idx]); vp != nil {
			vp.data.Store(key[idx+1:], val)
		}
	} else {
		v.data.Store(key, val)
	}
}

// GetString : Get string type config.
func (v *Viper) GetString(key string) string {
	return cast.ToString(v.Get(key))
}

// GetInt : Get int type config.
func (v *Viper) GetInt(key string) int {
	return cast.ToInt(v.Get(key))
}

// GetInt8 : Get int8 type config.
func (v *Viper) GetInt8(key string) int8 {
	return cast.ToInt8(v.Get(key))
}

// GetInt64 : Get int64 type config.
func (v *Viper) GetInt64(key string) int64 {
	return cast.ToInt64(v.Get(key))
}

// GetUint : Get uint type config.
func (v *Viper) GetUint(key string) uint {
	return cast.ToUint(v.Get(key))
}

// GetUint8 : Get uint8 type config.
func (v *Viper) GetUint8(key string) uint8 {
	return cast.ToUint8(v.Get(key))
}

// GetUint64 : Get uint64 type config.
func (v *Viper) GetUint64(key string) uint64 {
	return cast.ToUint64(v.Get(key))
}

// GetBool : Get bool type config.
func (v *Viper) GetBool(key string) bool {
	return cast.ToBool(v.Get(key))
}

// GetTime : Get time type config.
func (v *Viper) GetTime(key string) time.Time {
	return cast.ToTime(v.Get(key))
}

// GetDuration : Get duration type config.
func (v *Viper) GetDuration(key string) time.Duration {
	return cast.ToDuration(v.Get(key))
}

// GetIntSlice : Get int slice type config.
func (v *Viper) GetIntSlice(key string) []int {
	return cast.ToIntSlice(v.Get(key))
}

// GetStringSlice : Get string slice type config.
func (v *Viper) GetStringSlice(key string) []string {
	return cast.ToStringSlice(v.Get(key))
}

// GetStringMap : Get string map type config.
func (v *Viper) GetStringMap(key string) map[string]interface{} {
	return cast.ToStringMap(v.Get(key))
}

// GetStringMapString : Get string map string type config.
func (v *Viper) GetStringMapString(key string) map[string]string {
	return cast.ToStringMapString(v.Get(key))
}

// GetAddress : Get address type config.
func (v *Viper) GetAddress(key string) common.Address {
	return common.HexToAddress(v.GetString(key))
}

// GetBigInt : Get big.Int type config.
func (v *Viper) GetBigInt(key string) *big.Int {
	ret, failed := new(big.Int).SetString(v.GetString(key), 10)
	if !failed {
		ret = big.NewInt(0)
	}
	return ret
}

// GetECDSAKeys : Get ECDSA keys config.
func (v *Viper) GetECDSAKeys(key string) []*ecdsa.PrivateKey {
	keyLists := v.GetStringSlice(key)
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
