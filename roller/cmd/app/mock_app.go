package app

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	proverConfig "scroll-tech/prover/config"

	"scroll-tech/common/cmd"
	"scroll-tech/common/docker"
	"scroll-tech/common/utils"
)

var (
	proverIndex int
)

func getIndex() int {
	defer func() { proverIndex++ }()
	return proverIndex
}

// RollerApp prover-test client manager.
type RollerApp struct {
	Config *proverConfig.Config

	base *docker.App

	originFile string
	proverFile string
	bboltDB    string

	index int
	name  string
	args  []string
	docker.AppAPI
}

// NewRollerApp return a new proverApp manager.
func NewRollerApp(base *docker.App, file string, wsURL string) *RollerApp {
	proverFile := fmt.Sprintf("/tmp/%d_prover-config.json", base.Timestamp)
	proverApp := &RollerApp{
		base:       base,
		originFile: file,
		proverFile: proverFile,
		bboltDB:    fmt.Sprintf("/tmp/%d_bbolt_db", base.Timestamp),
		index:      getIndex(),
		name:       string(utils.RollerApp),
		args:       []string{"--log.debug", "--config", proverFile},
	}
	if err := proverApp.MockConfig(true, wsURL); err != nil {
		panic(err)
	}
	return proverApp
}

// RunApp run prover-test child process by multi parameters.
func (r *RollerApp) RunApp(t *testing.T, args ...string) {
	r.AppAPI = cmd.NewCmd(r.name, append(r.args, args...)...)
	r.AppAPI.RunApp(func() bool { return r.AppAPI.WaitResult(t, time.Second*40, "prover start successfully") })
}

// Free stop and release prover-test.
func (r *RollerApp) Free() {
	if !utils.IsNil(r.AppAPI) {
		r.AppAPI.WaitExit()
	}
	_ = os.Remove(r.proverFile)
	_ = os.Remove(r.Config.KeystorePath)
	_ = os.Remove(r.bboltDB)
}

// MockConfig creates a new prover config.
func (r *RollerApp) MockConfig(store bool, wsURL string) error {
	cfg, err := proverConfig.NewConfig(r.originFile)
	if err != nil {
		return err
	}
	cfg.RollerName = fmt.Sprintf("%s_%d", r.name, r.index)
	cfg.KeystorePath = fmt.Sprintf("/tmp/%d_%s.json", r.base.Timestamp, cfg.RollerName)
	cfg.TraceEndpoint = r.base.L2gethImg.Endpoint()
	// Reuse l1geth's keystore file
	cfg.KeystorePassword = "scrolltest"
	cfg.DBPath = r.bboltDB
	// Create keystore file.
	_, err = utils.LoadOrCreateKey(cfg.KeystorePath, cfg.KeystorePassword)
	if err != nil {
		return err
	}
	cfg.CoordinatorURL = wsURL
	r.Config = cfg

	if !store {
		return nil
	}

	data, err := json.Marshal(r.Config)
	if err != nil {
		return err
	}
	return os.WriteFile(r.proverFile, data, 0600)
}

// RollerApps proverApp list.
type RollerApps []*RollerApp

// RunApps starts all the proverApps.
func (r RollerApps) RunApps(t *testing.T, args ...string) {
	var eg errgroup.Group
	for i := range r {
		i := i
		eg.Go(func() error {
			r[i].RunApp(t, args...)
			return nil
		})
	}
	_ = eg.Wait()
}

// MockConfigs creates all the proverApps' configs.
func (r RollerApps) MockConfigs(store bool, wsURL string) error {
	var eg errgroup.Group
	for _, prover := range r {
		prover := prover
		eg.Go(func() error {
			return prover.MockConfig(store, wsURL)
		})
	}
	return eg.Wait()
}

// Free releases proverApps.
func (r RollerApps) Free() {
	var wg sync.WaitGroup
	wg.Add(len(r))
	for i := range r {
		i := i
		go func() {
			r[i].Free()
			wg.Done()
		}()
	}
	wg.Wait()
}

// WaitExit wait proverApps stopped.
func (r RollerApps) WaitExit() {
	var wg sync.WaitGroup
	wg.Add(len(r))
	for i := range r {
		i := i
		go func() {
			r[i].WaitExit()
			wg.Done()
		}()
	}
	wg.Wait()
}
