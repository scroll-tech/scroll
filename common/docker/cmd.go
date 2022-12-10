package docker

import (
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
func (tt *Cmd) RunApp(parallel bool) {
	tt.mu.Lock()
	defer tt.mu.Unlock()
	tt.Log("cmd: ", append([]string{tt.name}, tt.args...))
	tt.cmd = &exec.Cmd{
		Path:   reexec.Self(),
		Args:   append([]string{tt.name}, tt.args...),
		Stderr: tt,
		Stdout: tt,
	}
	if parallel {
		go func() { _ = tt.cmd.Run() }()
	} else {
		_ = tt.cmd.Run()
	}
}

// WaitExit wait util process exit.
func (tt *Cmd) WaitExit() {
	tt.mu.Lock()
	defer tt.mu.Unlock()

	// Send interrupt signal.
	_ = tt.cmd.Process.Signal(os.Interrupt)

	// Wait all the check funcs are finished or test status is failed.
	tick := time.NewTicker(time.Millisecond * 500)
	for !(tt.Failed() || tt.checkFuncs.IsEmpty()) {
		<-tick.C
	}
}

// Interrupt send interrupt signal.
func (tt *Cmd) Interrupt() {
	tt.mu.Lock()
	defer tt.mu.Unlock()
	tt.Err = tt.cmd.Process.Signal(os.Interrupt)
}

// RegistFunc register check func
func (tt *Cmd) RegistFunc(key string, check checkFunc) {
	tt.checkFuncs.Set(key, check)
}

// UnRegistFunc unregister check func
func (tt *Cmd) UnRegistFunc(key string) {
	tt.checkFuncs.Pop(key)
}

// ExpectWithTimeout wait result during timeout time.
func (tt *Cmd) ExpectWithTimeout(parallel bool, timeout time.Duration, keyword string) {
	tt.mu.Lock()
	defer tt.mu.Unlock()
	if keyword == "" {
		return
	}
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

	waitResult := func() {
		defer tt.UnRegistFunc(keyword)
		select {
		case <-okCh:
			return
		case <-time.After(timeout):
			assert.Failf(tt, "didn't get the desired result before timeout, keyword: %s", keyword)
		}
	}

	if parallel {
		go waitResult()
	} else {
		waitResult()
	}
}

func (tt *Cmd) runCmd() {
	cmd := exec.Command(tt.args[0], tt.args[1:]...) //nolint:gosec
	cmd.Stdout = tt
	cmd.Stderr = tt
	_ = cmd.Run()
}

// RunCmd parallel running when parallel is true.
func (tt *Cmd) RunCmd(parallel bool) {
	tt.Log("cmd: ", tt.args)
	if parallel {
		go tt.runCmd()
	} else {
		tt.runCmd()
	}
}

func (tt *Cmd) Write(data []byte) (int, error) {
	out := string(data)
	if verbose {
		tt.Logf("%s: %v", tt.name, out)
	} else if strings.Contains(out, "error") || strings.Contains(out, "warning") {
		tt.Logf("%s: %v", tt.name, out)
	}
	go tt.checkFuncs.IterCb(func(_ string, value interface{}) {
		check := value.(checkFunc)
		check(out)
	})
	return len(data), nil
}
