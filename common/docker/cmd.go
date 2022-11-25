package docker

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/pkg/reexec"
	"github.com/stretchr/testify/assert"
)

type checkFunc func(buf string)

// Cmd struct
type Cmd struct {
	*testing.T

	cmd     *exec.Cmd
	verbose bool

	checkFuncs sync.Map //map[string]checkFunc

	//stdout bytes.Buffer
	Err error

	stopCh chan struct{}
}

// NewCmd create Cmd instance.
func NewCmd(t *testing.T) *Cmd {
	return &Cmd{T: t, stopCh: make(chan struct{})}
}

// OpenLog set log open or close.
func (tt *Cmd) OpenLog(open bool) {
	tt.verbose = open
}

// Run exec's the current binary using name as argv[0] which will trigger the
// reexec init function for that name (e.g. "geth-test" in cmd/geth/run_test.go)
func (tt *Cmd) Run(name string, args ...string) {
	tt.cmd = &exec.Cmd{
		Path:   reexec.Self(),
		Args:   append([]string{name}, args...),
		Stderr: tt,
		Stdout: tt,
	}
	if err := tt.cmd.Run(); err != nil {
		tt.Fatal(err)
	}
}

// WaitExit wait util process exit.
func (tt *Cmd) WaitExit() {
	tt.Err = tt.cmd.Wait()
	select {
	case tt.stopCh <- struct{}{}:
	default:
	}
}

// Interrupt send interrupt signal.
func (tt *Cmd) Interrupt() {
	tt.Err = tt.cmd.Process.Signal(os.Interrupt)
}

// RegistFunc register check func
func (tt *Cmd) RegistFunc(key string, check checkFunc) {
	tt.checkFuncs.Store(key, check)
}

// UnRegistFunc unregister check func
func (tt *Cmd) UnRegistFunc(key string) {
	if _, ok := tt.checkFuncs.Load(key); ok {
		tt.checkFuncs.Delete(key)
	}
}

// ExpectWithTimeout wait result during timeout time.
func (tt *Cmd) ExpectWithTimeout(parallel bool, timeout time.Duration, keyword string) {
	okCh := make(chan struct{}, 1)
	tt.RegistFunc(keyword, func(buf string) {
		if strings.Contains(buf, keyword) {
			select {
			case okCh <- struct{}{}:
			default:
				return
			}
		}
	})

	//Wait result func.
	waitResult := func() {
		defer tt.UnRegistFunc(keyword)
		select {
		case <-okCh:
			return
		case <-tt.stopCh:
			assert.Error(tt, fmt.Errorf("didn't get the desired result before cmd stoped, keyword: %s", keyword))
		case <-time.After(timeout):
			assert.Error(tt, fmt.Errorf("didn't get the desired result before timeout, keyword: %s", keyword))
		}
	}

	if parallel {
		go waitResult()
	} else {
		waitResult()
	}
}

func (tt *Cmd) runCmd(args []string) {
	cmd := exec.Command(args[0], args[1:]...) //nolint:gosec
	cmd.Stdout = tt
	cmd.Stderr = tt
	_ = cmd.Run()
}

// RunCmd parallel running when parallel is true.
func (tt *Cmd) RunCmd(args []string, parallel bool) {
	tt.Log("RunCmd cmd", args)
	if parallel {
		go tt.runCmd(args)
	} else {
		tt.runCmd(args)
	}
}

func (tt *Cmd) Write(data []byte) (int, error) {
	out := string(data)
	if tt.verbose {
		tt.Logf(out)
	} else if strings.Contains(out, "error") || strings.Contains(out, "warning") {
		tt.Logf(out)
	}
	go tt.checkFuncs.Range(func(_, value interface{}) bool {
		check := value.(checkFunc)
		check(out)
		return true
	})
	return len(data), nil
}
