package docker

import (
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

var (
	verbose bool
	// Add this lock because of `os/exec` don't support parallel running.
	cmdMu sync.Mutex
)

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

	checkFuncs sync.Map //map[string]checkFunc
}

// NewCmd create Cmd instance.
func NewCmd(t *testing.T, name string, args ...string) *Cmd {
	cmd := &Cmd{
		T:    t,
		name: name,
		args: args,
	}
	return cmd
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
	go func(content string) {
		t.checkFuncs.Range(func(key, value any) bool {
			check := value.(checkFunc)
			check(content)
			return true
		})
	}(out)
	return len(data), nil
}

func (t *Cmd) runCmd() {
	cmd := exec.Command(t.args[0], t.args[1:]...) //nolint:gosec
	cmd.Stdout = t
	cmd.Stderr = t
	cmdMu.Lock()
	_ = cmd.Run()
	cmdMu.Unlock()
}
