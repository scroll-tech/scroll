package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/pkg/reexec"
	cmap "github.com/orcaman/concurrent-map"
	"github.com/stretchr/testify/assert"
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

	name string
	args []string

	mu  sync.Mutex
	cmd *exec.Cmd

	checkFuncs cmap.ConcurrentMap //map[string]checkFunc

	//stdout bytes.Buffer
	Err error
}

// NewCmd create Cmd instance.
func NewCmd(t *testing.T, name string, args ...string) *Cmd {
	return &Cmd{
		T:          t,
		checkFuncs: cmap.New(),
		name:       name,
		args:       args,
	}
}

// RunApp exec's the current binary using name as argv[0] which will trigger the
// reexec init function for that name (e.g. "geth-test" in cmd/geth/run_test.go)
func (t *Cmd) RunApp(parallel bool) {
	t.Log("cmd: ", append([]string{t.name}, t.args...))
	cmd := &exec.Cmd{
		Path:   reexec.Self(),
		Args:   append([]string{t.name}, t.args...),
		Stderr: t,
		Stdout: t,
	}
	if parallel {
		go func() {
			_ = cmd.Run()
		}()
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
	t.mu.Unlock()
}

// Interrupt send interrupt signal.
func (t *Cmd) Interrupt() {
	t.mu.Lock()
	t.Err = t.cmd.Process.Signal(os.Interrupt)
	t.mu.Unlock()
}

// RegistFunc register check func
func (t *Cmd) RegistFunc(key string, check checkFunc) {
	t.checkFuncs.Set(key, check)
}

// UnRegistFunc unregister check func
func (t *Cmd) UnRegistFunc(key string) {
	t.checkFuncs.Pop(key)
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

func (t *Cmd) runCmd() {
	cmd := exec.Command(t.args[0], t.args[1:]...) //nolint:gosec
	cmd.Stdout = t
	cmd.Stderr = t
	_ = cmd.Run()
}

// RunCmd parallel running when parallel is true.
func (t *Cmd) RunCmd(parallel bool) {
	t.Log("cmd: ", t.args)
	if parallel {
		go t.runCmd()
	} else {
		t.runCmd()
	}
}

func (t *Cmd) Write(data []byte) (int, error) {
	out := string(data)
	if verbose {
		t.Logf("%s: %v", t.name, out)
	} else if strings.Contains(out, "error") || strings.Contains(out, "warning") {
		t.Logf("%s: %v", t.name, out)
	}
	go t.checkFuncs.IterCb(func(_ string, value interface{}) {
		check := value.(checkFunc)
		check(out)
	})
	return len(data), nil
}
