package integration

import (
	"testing"
	"time"

	_ "scroll-tech/bridge/cmd/app"
	_ "scroll-tech/coordinator/cmd/app"
	_ "scroll-tech/database/cmd/app"
	_ "scroll-tech/roller/cmd/app"
)

var (
	base        *dockerApp
	bridge      *bridgeApp
	coordinator *coordinatorApp
	rollers     rollerApps
)

type appAPI interface {
	WaitResult(t *testing.T, timeout time.Duration, keyword string) bool
	RunApp(waitResult func() bool)
	WaitExit()
	ExpectWithTimeout(t *testing.T, parallel bool, timeout time.Duration, keyword string)
}
