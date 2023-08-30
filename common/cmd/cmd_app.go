package cmd

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// IsRunning 1 started, 0 not started.
func (c *Cmd) IsRunning() bool {
	return atomic.LoadUint64(&c.isRunning) == 1
}

func (c *Cmd) runApp() {
	fmt.Println("cmd:", append([]string{c.name}, c.args...))
	if atomic.CompareAndSwapUint64(&c.isRunning, 0, 1) {
		c.ErrChan <- c.app.Run()
	}
}

// RunApp exec's the current binary using name as argv[0] which will trigger the
// reexec init function for that name (e.g. "geth-test" in cmd/geth/run_test.go)
func (c *Cmd) RunApp(waitResult func() bool) {
	if waitResult != nil {
		go func() {
			c.runApp()
		}()
		waitResult()
	} else {
		c.runApp()
	}
}

// WaitExit wait util process exit.
func (c *Cmd) WaitExit() {
	if atomic.LoadUint64(&c.isRunning) == 0 {
		return
	}
	// Wait all the check functions are finished, interrupt loop when appear error.
	var err error
	for err == nil && !c.checkFuncs.IsEmpty() {
		select {
		case err = <-c.ErrChan:
			if err != nil {
				fmt.Printf("%s appear error durning running, err: %v\n", c.name, err)
			}
		default:
			<-time.After(time.Millisecond * 500)
		}
	}

	// Send interrupt signal.
	_ = c.cmd.Process.Signal(os.Interrupt)
	// should use `_ = c.cmd.Process.Wait()` here, but we have some bugs in coordinator's graceful exit,
	// so we use `Kill` as a temp workaround. And since `WaitExit` is only used in integration tests, so
	// it won't really affect our functionalities.
	_ = c.cmd.Process.Kill()
}

// Interrupt send interrupt signal.
func (c *Cmd) Interrupt() {
	c.ErrChan <- c.cmd.Process.Signal(os.Interrupt)
}

// WaitResult return true when get the keyword during timeout.
func (c *Cmd) WaitResult(t *testing.T, timeout time.Duration, keyword string) bool {
	if keyword == "" {
		return false
	}
	okCh := make(chan struct{}, 1)
	c.RegistFunc(keyword, func(buf string) {
		if strings.Contains(buf, keyword) {
			select {
			case okCh <- struct{}{}:
			default:
				return
			}
		}
	})
	defer c.UnRegistFunc(keyword)
	select {
	case <-okCh:
		return true
	case <-time.After(timeout):
		assert.Fail(t, fmt.Sprintf("didn't get the desired result before timeout, keyword: %s", keyword))
	}
	return false
}

// ExpectWithTimeout wait result during timeout time.
func (c *Cmd) ExpectWithTimeout(t *testing.T, parallel bool, timeout time.Duration, keyword string) {
	if keyword == "" {
		return
	}
	okCh := make(chan struct{}, 1)
	c.RegistFunc(keyword, func(buf string) {
		if strings.Contains(buf, keyword) {
			select {
			case okCh <- struct{}{}:
			default:
				return
			}
		}
	})

	waitResult := func() {
		defer c.UnRegistFunc(keyword)
		select {
		case <-okCh:
			return
		case <-time.After(timeout):
			assert.Fail(t, fmt.Sprintf("didn't get the desired result before timeout, keyword: %s", keyword))
		}
	}

	if parallel {
		go waitResult()
	} else {
		waitResult()
	}
}
