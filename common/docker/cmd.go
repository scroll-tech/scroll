package docker

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
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

	checkFuncs sync.Map //map[string]checkFunc

	//stdout bytes.Buffer
	errMsg chan error
}

// NewCmd create Cmd instance.
func NewCmd(t *testing.T) *Cmd {
	cmd := &Cmd{
		T: t,
		//stdout:   bytes.Buffer{},
		errMsg: make(chan error, 2),
	}
	// Handle panic.
	cmd.RegistFunc("panic", func(buf string) {
		if strings.Contains(buf, "panic") {
			cmd.errMsg <- errors.New(buf)
		}
	})
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
func (t *Cmd) RunCmd(args []string, parallel bool) {
	t.Log("RunCmd cmd", args)
	if parallel {
		go t.runCmd(args)
	} else {
		t.runCmd(args)
	}
}

// ErrMsg return error output channel
func (t *Cmd) ErrMsg() <-chan error {
	return t.errMsg
}

func (t *Cmd) Write(data []byte) (int, error) {
	out := string(data)
	if verbose {
		t.Logf(out)
	} else if strings.Contains(out, "error") || strings.Contains(out, "warning") {
		t.Logf(out)
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

func (t *Cmd) runCmd(args []string) {
	cmd := exec.Command(args[0], args[1:]...) //nolint:gosec
	cmd.Stdout = t
	cmd.Stderr = t
	_ = cmd.Run()
}
