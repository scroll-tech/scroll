package app

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/rpc"

	"scroll-tech/prover/config"

	"scroll-tech/common/cmd"
	"scroll-tech/common/docker"
	"scroll-tech/common/testcontainers"
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
	Config *config.Config

	testApps *testcontainers.TestcontainerApps

	originFile string
	proverFile string
	bboltDB    string

	index int
	name  string
	args  []string
	docker.AppAPI
}

// NewProverApp return a new proverApp manager.
func NewProverApp(testApps *testcontainers.TestcontainerApps, mockName utils.MockAppName, file string, httpURL string) *ProverApp {
	var proofType message.ProofType
	switch mockName {
	case utils.ChunkProverApp:
		proofType = message.ProofTypeChunk
	case utils.BatchProverApp:
		proofType = message.ProofTypeBatch
	default:
		return nil
	}
	name := string(mockName)
	proverFile := fmt.Sprintf("/tmp/%d_%s-config.json", testApps.Timestamp, name)
	proverApp := &ProverApp{
		testApps:   testApps,
		originFile: file,
		proverFile: proverFile,
		bboltDB:    fmt.Sprintf("/tmp/%d_%s_bbolt_db", testApps.Timestamp, name),
		index:      getIndex(),
		name:       name,
		args:       []string{"--log.debug", "--config", proverFile},
	}
	proverApp.AppAPI = cmd.NewCmd(proverApp.name, proverApp.args...)
	if err := proverApp.MockConfig(true, httpURL, proofType); err != nil {
		panic(err)
	}
	return proverApp
}

// RunApp run prover-test child process by multi parameters.
func (r *ProverApp) RunApp(t *testing.T) {
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
	cfg, err := config.NewConfig(r.originFile)
	if err != nil {
		return err
	}
	cfg.ProverName = fmt.Sprintf("%s_%d", r.name, r.index)
	cfg.KeystorePath = fmt.Sprintf("/tmp/%d_%s.json", r.testApps.Timestamp, cfg.ProverName)

	endpoint, err := r.testApps.GetL2GethEndPoint()
	if err != nil {
		return err
	}
	cfg.L2Geth.Endpoint = endpoint
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
