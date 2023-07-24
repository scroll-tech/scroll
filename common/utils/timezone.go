package utils

import "time"

// NowUTC get the utc time.Now
func NowUTC() time.Time {
	utc, _ := time.LoadLocation("")
	return time.Now().In(utc)
}
