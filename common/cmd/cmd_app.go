package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/docker/docker/pkg/reexec"
	"github.com/stretchr/testify/assert"
)

// RunApp exec's the current binary using name as argv[0] which will trigger the
// reexec init function for that name (e.g. "geth-test" in cmd/geth/run_test.go)
func (t *Cmd) RunApp(waitResult func() bool) {
	t.Log("cmd: ", append([]string{t.name}, t.args...))
	cmd := &exec.Cmd{
		Path:   reexec.Self(),
		Args:   append([]string{t.name}, t.args...),
		Stderr: t,
		Stdout: t,
	}
	if waitResult != nil {
		go func() {
			_ = cmd.Run()
		}()
		waitResult()
	} else {
		_ = cmd.Run()
	}

	t.mu.Lock()
	t.cmd = cmd
	t.mu.Unlock()
}

// WaitExit wait util process exit.
func (t *Cmd) WaitExit() {
	// Wait all the check funcs are finished or test status is failed.
	for !(t.Failed() || t.checkFuncs.IsEmpty()) {
		<-time.After(time.Millisecond * 500)
	}

	// Send interrupt signal.
	t.mu.Lock()
	_ = t.cmd.Process.Signal(os.Interrupt)
	_, _ = t.cmd.Process.Wait()
	t.mu.Unlock()
}

// Interrupt send interrupt signal.
func (t *Cmd) Interrupt() {
	t.mu.Lock()
	t.Err = t.cmd.Process.Signal(os.Interrupt)
	t.mu.Unlock()
}

// WaitResult return true when get the keyword during timeout.
func (t *Cmd) WaitResult(timeout time.Duration, keyword string) bool {
	if keyword == "" {
		return false
	}
	okCh := make(chan struct{}, 1)
	t.RegistFunc(keyword, func(buf string) {
		if strings.Contains(buf, keyword) {
			select {
			case okCh <- struct{}{}:
			default:
				return
			}
		}
	})
	defer t.UnRegistFunc(keyword)
	select {
	case <-okCh:
		return true
	case <-time.After(timeout):
		assert.Fail(t, fmt.Sprintf("didn't get the desired result before timeout, keyword: %s", keyword))
	}
	return false
}

// ExpectWithTimeout wait result during timeout time.
func (t *Cmd) ExpectWithTimeout(parallel bool, timeout time.Duration, keyword string) {
	if keyword == "" {
		return
	}
	okCh := make(chan struct{}, 1)
	t.RegistFunc(keyword, func(buf string) {
		if strings.Contains(buf, keyword) {
			select {
			case okCh <- struct{}{}:
			default:
				return
			}
		}
	})

	waitResult := func() {
		defer t.UnRegistFunc(keyword)
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
