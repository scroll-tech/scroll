package utils

import (
	"math/big"
	"time"
)

var (
	Ether = big.NewInt(1e18)
)

// TryTimes try run several times until the function return true.
func TryTimes(times int, run func() bool) {
	for i := 0; i < times; i++ {
		if run() {
			return
		}
		time.Sleep(time.Millisecond * 500)
	}
}
