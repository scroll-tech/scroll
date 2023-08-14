package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/pkg/reexec"
	"github.com/stretchr/testify/assert"
)

// RunApp exec's the current binary using name as argv[0] which will trigger the
// reexec init function for that name (e.g. "geth-test" in cmd/geth/run_test.go)
func (c *Cmd) RunApp(waitResult func() bool) {
	fmt.Println("cmd: ", append([]string{c.name}, c.args...))
	cmd := &exec.Cmd{
		Path:   reexec.Self(),
		Args:   append([]string{c.name}, c.args...),
		Stderr: c,
		Stdout: c,
	}
	if waitResult != nil {
		go func() {
			_ = cmd.Run()
		}()
		waitResult()
	} else {
		_ = cmd.Run()
	}

	c.mu.Lock()
	c.cmd = cmd
	c.mu.Unlock()
}

// WaitExit wait util process exit.
func (c *Cmd) WaitExit() {
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
	c.mu.Lock()
	if c.cmd != nil {
		_ = c.cmd.Process.Signal(os.Interrupt)
		// should use `_ = c.cmd.Process.Wait()` here, but we have some bugs in coordinator's graceful exit,
		// so we use `Kill` as a temp workaround. And since `WaitExit` is only used in integration tests, so
		// it won't really affect our functionalities.
		_ = c.cmd.Process.Kill()
	}
	c.mu.Unlock()
}

// Interrupt send interrupt signal.
func (c *Cmd) Interrupt() {
	c.mu.Lock()
	c.ErrChan <- c.cmd.Process.Signal(os.Interrupt)
	c.mu.Unlock()
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
