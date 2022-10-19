package utils

import "os"

// GetEnvWithDefault get value from env if is none use the default
func GetEnvWithDefault(key string, defult string) string {
	val := os.Getenv(key)
	if len(val) == 0 {
		val = defult
	}
	return val
}
