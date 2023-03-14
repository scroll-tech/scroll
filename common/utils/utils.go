package utils

import (
	"context"
	"time"
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

// LoopWithContext Run the f func with context periodically.
func LoopWithContext(ctx context.Context, second time.Duration, f func(ctx context.Context)) {
	tick := time.NewTicker(second)
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
func Loop(ctx context.Context, second time.Duration, f func()) {
	tick := time.NewTicker(second)
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
