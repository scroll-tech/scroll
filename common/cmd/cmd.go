package cmd

import (
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"

	cmap "github.com/orcaman/concurrent-map"
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

// RegistFunc register check func
func (t *Cmd) RegistFunc(key string, check checkFunc) {
	t.checkFuncs.Set(key, check)
}

// UnRegistFunc unregister check func
func (t *Cmd) UnRegistFunc(key string) {
	t.checkFuncs.Pop(key)
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
