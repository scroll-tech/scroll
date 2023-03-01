package utils

import "time"

// TryTimes try run several times until the function return true.
func TryTimes(times int, run func() bool) {
	for i := 0; times == -1 || i < times; i++ {
		if run() {
			return
		}
		time.Sleep(time.Millisecond * 500)
	}
}
