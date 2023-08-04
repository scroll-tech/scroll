package app

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/scroll-tech/go-ethereum/rpc"
	"golang.org/x/sync/errgroup"

	proverConfig "scroll-tech/prover/config"

	"scroll-tech/common/cmd"
	"scroll-tech/common/docker"
	"scroll-tech/common/types/message"
	"scroll-tech/common/utils"
)

var (
	proverIndex int
)

func getIndex() int {
	defer func() { proverIndex++ }()
	return proverIndex
}

// ProverApp prover-test client manager.
type ProverApp struct {
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

// NewProverApp return a new proverApp manager.
func NewProverApp(base *docker.App, file string, httpURL string, proofType message.ProofType) *ProverApp {
	uuid := uuid.New().String()

	proverFile := fmt.Sprintf("/tmp/%s_%d_prover-config.json", uuid, base.Timestamp)
	proverApp := &ProverApp{
		base:       base,
		originFile: file,
		proverFile: proverFile,
		bboltDB:    fmt.Sprintf("/tmp/%s_%d_bbolt_db", uuid, base.Timestamp),
		index:      getIndex(),
		name:       string(utils.ProverApp),
		args:       []string{"--log.debug", "--config", proverFile},
	}
	if err := proverApp.MockConfig(true, httpURL, proofType); err != nil {
		panic(err)
	}
	return proverApp
}

// RunApp run prover-test child process by multi parameters.
func (r *ProverApp) RunApp(t *testing.T, args ...string) {
	r.AppAPI = cmd.NewCmd(r.name, append(r.args, args...)...)
	r.AppAPI.RunApp(func() bool { return r.AppAPI.WaitResult(t, time.Second*40, "prover start successfully") })
}

// Free stop and release prover-test.
func (r *ProverApp) Free() {
	if !utils.IsNil(r.AppAPI) {
		r.AppAPI.WaitExit()
	}
	_ = os.Remove(r.proverFile)
	_ = os.Remove(r.Config.KeystorePath)
	_ = os.Remove(r.bboltDB)
}

// MockConfig creates a new prover config.
func (r *ProverApp) MockConfig(store bool, httpURL string, proofType message.ProofType) error {
	cfg, err := proverConfig.NewConfig(r.originFile)
	if err != nil {
		return err
	}
	cfg.ProverName = fmt.Sprintf("%s_%d", r.name, r.index)
	cfg.KeystorePath = fmt.Sprintf("/tmp/%d_%s.json", r.base.Timestamp, cfg.ProverName)
	cfg.L2Geth.Endpoint = r.base.L2gethImg.Endpoint()
	cfg.L2Geth.Confirmations = rpc.LatestBlockNumber
	// Reuse l1geth's keystore file
	cfg.KeystorePassword = "scrolltest"
	cfg.DBPath = r.bboltDB
	// Create keystore file.
	_, err = utils.LoadOrCreateKey(cfg.KeystorePath, cfg.KeystorePassword)
	if err != nil {
		return err
	}
	cfg.Coordinator.BaseURL = httpURL
	cfg.Coordinator.RetryCount = 10
	cfg.Coordinator.RetryWaitTimeSec = 10
	cfg.Coordinator.ConnectionTimeoutSec = 30
	cfg.Core.ProofType = proofType
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

// ProverApps proverApp list.
type ProverApps []*ProverApp

// RunApps starts all the proverApps.
func (r ProverApps) RunApps(t *testing.T, args ...string) {
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

// Free releases proverApps.
func (r ProverApps) Free() {
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
func (r ProverApps) WaitExit() {
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
