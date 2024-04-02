package docker

import (
	"testing"
	"time"
)

// AppAPI app interface.
type AppAPI interface {
	IsRunning() bool
	WaitResult(t *testing.T, timeout time.Duration, keyword string) bool
	RunApp(waitResult func() bool)
	WaitExit()
	ExpectWithTimeout(t *testing.T, parallel bool, timeout time.Duration, keyword string)
}
