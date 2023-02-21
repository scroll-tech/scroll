package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	rollerConfig "scroll-tech/roller/config"

	"scroll-tech/common/cmd"
)

var (
	rollerIndex int
)

func getIndex() int {
	rollerIndex++
	return rollerIndex
}

type rollerApp struct {
	base *dockerApp

	cfg *rollerConfig.Config

	rollerFile string
	bboltDB    string
	keystore   string

	index int
	name  string
	args  []string
	appAPI
}

func newRollerApp(base *dockerApp) *rollerApp {
	file := fmt.Sprintf("/tmp/%d_roller-config.json", base.timestamp)
	app := &rollerApp{
		base:       base,
		rollerFile: file,
		bboltDB:    fmt.Sprintf("/tmp/%d_bbolt_db", base.timestamp),
		index:      getIndex(),
		name:       "roller-test",
		args:       []string{"--log.debug", "--config", file},
	}
	return app
}

func (r *rollerApp) runApp(t *testing.T, args ...string) {
	// Reset roller config file.
	if err := r.mockRollerConfig(); err != nil {
		t.Fatal(err)
	}
	r.appAPI = cmd.NewCmd(t, r.name, append(args, r.args...)...)
	r.appAPI.RunApp(func() bool { return r.appAPI.WaitResult(time.Second*40, "roller start successfully") })
}

func (r *rollerApp) free() {
	_ = os.Remove(r.rollerFile)
	_ = os.Remove(r.bboltDB)
}

func (r *rollerApp) mockRollerConfig() error {
	if r.cfg == nil {
		cfg, err := rollerConfig.NewConfig("../../roller/config.json")
		if err != nil {
			return err
		}
		cfg.RollerName = fmt.Sprintf("%s_%d", r.name, r.index)
		cfg.KeystorePath = fmt.Sprintf("/tmp/%d_%s.json", r.base.timestamp, cfg.RollerName)
		// Reuse l1geth's keystore file
		cfg.KeystorePassword = "scrolltest"
		cfg.DBPath = r.bboltDB
		r.cfg = cfg
	}
	// Clean db files and reset them.
	r.free()

	r.cfg.CoordinatorURL = fmt.Sprintf("ws://localhost:%d", r.base.wsPort)

	data, err := json.Marshal(r.cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(r.rollerFile, data, 0644)
}

type rollerApps []*rollerApp

func (r rollerApps) runApps(t *testing.T, args ...string) {
	for _, roller := range r {
		roller.runApp(t, args...)
	}
}

func (r rollerApps) free() {
	for _, roller := range r {
		roller.free()
	}
}

func (r rollerApps) WaitExit() {
	for _, roller := range r {
		roller.WaitExit()
	}
}
