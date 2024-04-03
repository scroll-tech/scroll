package cmd

import (
    "fmt"
    "os"
    "os/exec"
    "strings"
    "sync"
    "sync/atomic"

    "github.com/docker/docker/pkg/reexec"
    cmap "github.com/orcaman/concurrent-map"
)

var verbose bool

func init() {
    v := os.Getenv("LOG_DOCKER")
    if strings.ToLower(v) == "true" {
        verbose = true
    }
}

type checkFunc func(buf string)

// Cmd struct
type Cmd struct {
    name       string
    args       []string
    isRunning  uint64
    cmd        *exec.Cmd
    app        *exec.Cmd
    checkFuncs cmap.ConcurrentMap //map[string]checkFunc
    // open log flag.
    openLog bool
    // error channel
    ErrChan chan error
    // wait group for parallel runs
    wg sync.WaitGroup
}

// NewCmd create Cmd instance.
func NewCmd(name string, params ...string) *Cmd {
    cmd := &Cmd{
        checkFuncs: cmap.New(),
        name:       name,
        args:       params,
        ErrChan:    make(chan error, 10),
        cmd:        exec.Command(name, params...),
        app: &exec.Cmd{
            Path: reexec.Self(),
            Args: append([]string{name}, params...),
        },
    }

    cmd.cmd.Stdout = cmd
    cmd.cmd.Stderr = cmd
    cmd.app.Stdout = cmd
    cmd.app.Stderr = cmd

    return cmd
}

// RegisterFunc register check func
func (c *Cmd) RegisterFunc(key string, check checkFunc) {
    c.checkFuncs.Set(key, check)
}

// UnregisterFunc unregister check func
func (c *Cmd) UnregisterFunc(key string) {
    c.checkFuncs.Pop(key)
}

func (c *Cmd) runCmd() {
    fmt.Println("cmd:", append([]string{c.name}, c.args...))
    if atomic.CompareAndSwapUint64(&c.isRunning, 0, 1) {
        c.ErrChan <- c.cmd.Run()
    }
}

// RunCmd parallel running when parallel is true.
func (c *Cmd) RunCmd(parallel bool) {
    if parallel {
        c.wg.Add(1)
        go func() {
            defer c.wg.Done()
            c.runCmd()
        }()
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
    } else if strings.ContainsAny(strings.ToLower(out), "error") ||
        strings.ContainsAny(strings.ToLower(out), "warning") {
        fmt.Printf("%s:\n\t%v", c.name, out)
    }

    go c.checkFuncs.IterCb(func(_ string, value interface{}) {
        check := value.(checkFunc)
        check(out)
    })

    return len(data), nil
}
