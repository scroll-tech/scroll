package app

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/modern-go/reflect2"

	"scroll-tech/common/cmd"
	"scroll-tech/common/docker"
	"scroll-tech/common/utils"
	rollerConfig "scroll-tech/roller/config"
)

var (
	rollerIndex int
)

func getIndex() int {
	rollerIndex++
	return rollerIndex
}

type RollerApp struct {
	base *docker.DockerApp

	cfg        *rollerConfig.Config
	originFile string
	rollerFile string
	bboltDB    string
	keystore   string

	index int
	name  string
	args  []string
	docker.AppAPI
}

func NewRollerApp(base *docker.DockerApp, wsUrl string, file string) *RollerApp {
	rollerFile := fmt.Sprintf("/tmp/%d_roller-config.json", base.Timestamp)
	rollerApp := &RollerApp{
		base:       base,
		originFile: file,
		rollerFile: rollerFile,
		bboltDB:    fmt.Sprintf("/tmp/%d_bbolt_db", base.Timestamp),
		index:      getIndex(),
		name:       "roller-test",
		args:       []string{"--log.debug", "--config", rollerFile},
	}
	if err := rollerApp.MockRollerConfig(wsUrl); err != nil {
		panic(err)
	}

	return rollerApp
}

func (r *RollerApp) RunApp(t *testing.T, args ...string) {
	r.AppAPI = cmd.NewCmd(r.name, append(r.args, args...)...)
	r.AppAPI.RunApp(func() bool { return r.AppAPI.WaitResult(t, time.Second*40, "roller start successfully") })
}

func (r *RollerApp) Free() {
	if !reflect2.IsNil(r.AppAPI) {
		r.AppAPI.WaitExit()
		_ = os.Remove(r.rollerFile)
		_ = os.Remove(r.bboltDB)
	}
}

func (r *RollerApp) MockRollerConfig(wsUrl string) error {
	if r.cfg == nil {
		cfg, err := rollerConfig.NewConfig(r.originFile)
		if err != nil {
			return err
		}
		cfg.RollerName = fmt.Sprintf("%s_%d", r.name, r.index)
		cfg.KeystorePath = fmt.Sprintf("/tmp/%s_%d.json", cfg.RollerName, r.base.Timestamp)
		// Reuse l1geth's keystore file
		cfg.KeystorePassword = "scrolltest"
		cfg.DBPath = r.bboltDB
		// Create keystore file.
		_, err = utils.LoadOrCreateKey(cfg.KeystorePath, cfg.KeystorePassword)
		if err != nil {
			return err
		}
		r.cfg = cfg
	}

	r.cfg.CoordinatorURL = wsUrl

	data, err := json.Marshal(r.cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(r.rollerFile, data, 0644)
}

type RollerApps []*RollerApp

func (r RollerApps) RunApps(t *testing.T, args ...string) {
	for i := range r {
		r[i].RunApp(t, args...)
	}
}

func (r RollerApps) Free() {
	var wg sync.WaitGroup
	wg.Add(len(r))
	for i := range r {
		go func() {
			r[i].Free()
			wg.Done()
		}()
	}
	wg.Wait()
}

func (r RollerApps) WaitExit() {
	var wg sync.WaitGroup
	wg.Add(len(r))
	for i := range r {
		go func() {
			r[i].WaitExit()
			wg.Done()
		}()
	}
	wg.Wait()
}
