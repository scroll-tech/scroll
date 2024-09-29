package utils

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/modern-go/reflect2"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/log"
)

// TryTimes try run several times until the function return true.
func TryTimes(times int, run func() bool) bool {
	for i := 0; i < times; i++ {
		if run() {
			return true
		}
		time.Sleep(time.Millisecond * 500)
	}
	return false
}

// LoopWithContext Run the f func with context periodically.
func LoopWithContext(ctx context.Context, period time.Duration, f func(ctx context.Context)) {
	tick := time.NewTicker(period)
	defer tick.Stop()
	for ; ; <-tick.C {
		select {
		case <-ctx.Done():
			return
		default:
			f(ctx)
		}
	}
}

// Loop Run the f func periodically.
func Loop(ctx context.Context, period time.Duration, f func()) {
	tick := time.NewTicker(period)
	defer tick.Stop()
	for ; ; <-tick.C {
		select {
		case <-ctx.Done():
			return
		default:
			f()
		}
	}
}

// IsNil Check if the interface is empty.
func IsNil(i interface{}) bool {
	return i == nil || reflect2.IsNil(i)
}

// RandomURL return a random port endpoint.
func RandomURL() string {
	id, _ := rand.Int(rand.Reader, big.NewInt(5000-1))
	return fmt.Sprintf("localhost:%d", 10000+2000+id.Int64())
}

// ReadGenesis parses and returns the genesis file at the given path
func ReadGenesis(genesisPath string) (*core.Genesis, error) {
	file, err := os.Open(filepath.Clean(genesisPath))
	if err != nil {
		return nil, err
	}

	genesis := new(core.Genesis)
	if err := json.NewDecoder(file).Decode(genesis); err != nil {
		return nil, errors.Join(err, file.Close())
	}
	return genesis, file.Close()
}

// OverrideConfigWithEnv recursively overrides config values with environment variables
func OverrideConfigWithEnv(cfg interface{}, prefix string) error {
	v := reflect.ValueOf(cfg)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return nil
	}
	v = v.Elem()

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		tag := field.Tag.Get("json")
		if tag == "" {
			tag = strings.ToLower(field.Name)
		}

		envKey := prefix + "_" + strings.ToUpper(tag)

		switch fieldValue.Kind() {
		case reflect.Ptr:
			if !fieldValue.IsNil() {
				err := OverrideConfigWithEnv(fieldValue.Interface(), envKey)
				if err != nil {
					return err
				}
			}
		case reflect.Struct:
			err := OverrideConfigWithEnv(fieldValue.Addr().Interface(), envKey)
			if err != nil {
				return err
			}
		default:
			if envValue, exists := os.LookupEnv(envKey); exists {
				log.Info("Overriding config with env var", "key", envKey)
				err := setField(fieldValue, envValue)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// setField sets the value of a field based on the environment variable value
func setField(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(intValue)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintValue, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(uintValue)
	case reflect.Float32, reflect.Float64:
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(floatValue)
	case reflect.Bool:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(boolValue)
	default:
		return fmt.Errorf("unsupported type: %v", field.Kind())
	}
	return nil
}
