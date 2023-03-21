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
	fmt.Printf("1. =============== %s %s\n", c.name, c.checkFuncs.Keys())
	// Wait all the check funcs are finished or test status is failed.
	for !(c.Err != nil || c.checkFuncs.IsEmpty()) {
		<-time.After(time.Second)
	}
	fmt.Printf("2. =============== %s %s\n", c.name, c.checkFuncs.Keys())

	// Send interrupt signal.
	c.mu.Lock()
	fmt.Printf("3. =============== %s %s\n", c.name, c.checkFuncs.Keys())
	_ = c.cmd.Process.Signal(os.Interrupt)
	fmt.Printf("4. =============== %s %s\n", c.name, c.checkFuncs.Keys())
	_ = c.cmd.Process.Kill()
	fmt.Printf("5. =============== %s %s\n", c.name, c.checkFuncs.Keys())
	c.mu.Unlock()
	fmt.Printf("6. =============== %s %s\n", c.name, c.checkFuncs.Keys())
}

// Interrupt send interrupt signal.
func (c *Cmd) Interrupt() {
	c.mu.Lock()
	c.Err = c.cmd.Process.Signal(os.Interrupt)
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
