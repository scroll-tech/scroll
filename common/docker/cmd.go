package docker

import (
	"bytes"
	"github.com/docker/docker/pkg/reexec"
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

var verbose bool

func init() {
	v := os.Getenv("LOG_DOCKER")
	if v == "true" || v == "TRUE" {
		verbose = true
	}
}

type checkFunc func(buf string)

// Cmd struct
type Cmd struct {
	*testing.T

	cmd *exec.Cmd
	buf bytes.Buffer

	checkFuncs sync.Map //map[string]checkFunc

	//stdout bytes.Buffer
	Err error
}

// NewCmd create Cmd instance.
func NewCmd(t *testing.T) *Cmd {
	return &Cmd{T: t}
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

func (tt *Cmd) WaitExit() {
	tt.Err = tt.cmd.Wait()
}

func (tt *Cmd) Interrupt() {
	tt.Err = tt.cmd.Process.Signal(os.Interrupt)
}

// RegistFunc register check func
func (t *Cmd) RegistFunc(key string, check checkFunc) {
	t.checkFuncs.Store(key, check)
}

// UnRegistFunc unregister check func
func (t *Cmd) UnRegistFunc(key string) {
	if _, ok := t.checkFuncs.Load(key); ok {
		t.checkFuncs.Delete(key)
	}
}

func (t *Cmd) ExpectWithTimeout(timeout time.Duration, keyword string) {
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

	go func() {
		select {
		case <-okCh:
			return
		case <-time.After(timeout):
			assert.Fail(t, "should have the keyword", keyword)
		}
	}()
}

func (t *Cmd) runCmd(args []string) {
	cmd := exec.Command(args[0], args[1:]...) //nolint:gosec
	cmd.Stdout = t
	cmd.Stderr = t
	_ = cmd.Run()
}

// RunCmd parallel running when parallel is true.
func (t *Cmd) RunCmd(args []string, parallel bool) {
	t.Log("RunCmd cmd", args)
	if parallel {
		go t.runCmd(args)
	} else {
		t.runCmd(args)
	}
}

func (t *Cmd) Write(data []byte) (int, error) {
	out := string(data)
	if verbose {
		t.Logf(out)
	} else if strings.Contains(out, "error") || strings.Contains(out, "warning") {
		t.Logf(out)
	}
	go t.checkFuncs.Range(func(key, value interface{}) bool {
		check := value.(checkFunc)
		check(out)
		return true
	})
	return len(data), nil
}
