package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

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
	name string
	args []string

	mu  sync.Mutex
	cmd *exec.Cmd

	checkFuncs cmap.ConcurrentMap //map[string]checkFunc

	// open log flag.
	openLog bool
	// error channel
	ErrChan chan error
}

// NewCmd create Cmd instance.
func NewCmd(name string, args ...string) *Cmd {
	return &Cmd{
		checkFuncs: cmap.New(),
		name:       name,
		args:       args,
		ErrChan:    make(chan error, 10),
	}
}

// RegistFunc register check func
func (c *Cmd) RegistFunc(key string, check checkFunc) {
	c.checkFuncs.Set(key, check)
}

// UnRegistFunc unregister check func
func (c *Cmd) UnRegistFunc(key string) {
	c.checkFuncs.Pop(key)
}

func (c *Cmd) runCmd() {
	cmd := exec.Command(c.args[0], c.args[1:]...) //nolint:gosec
	cmd.Stdout = c
	cmd.Stderr = c
	c.ErrChan <- cmd.Run()
}

// RunCmd parallel running when parallel is true.
func (c *Cmd) RunCmd(parallel bool) {
	fmt.Println("cmd:", c.args)
	if parallel {
		go c.runCmd()
	} else {
		c.runCmd()
	}
}

// OpenLog open cmd log by this api.
func (c *Cmd) OpenLog(open bool) {
	c.openLog = open
}

func (c *Cmd) Write(data []byte) (int, error) {
	out := string(data)
	if verbose || c.openLog {
		fmt.Printf("%s:\n\t%v", c.name, out)
	} else if strings.Contains(out, "error") || strings.Contains(out, "warning") {
		fmt.Printf("%s:\n\t%v", c.name, out)
	}
	go c.checkFuncs.IterCb(func(_ string, value interface{}) {
		check := value.(checkFunc)
		check(out)
	})
	return len(data), nil
}
