package coordinator_test

import (
	"context"
	"math/big"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"

	"scroll-tech/bridge/l2"
	"scroll-tech/bridge/sender"
	"scroll-tech/common/docker"
	"scroll-tech/common/message"
	"scroll-tech/common/utils"
	"scroll-tech/database"
	"scroll-tech/database/migrate"
	"scroll-tech/database/orm"

	"scroll-tech/coordinator"
	client2 "scroll-tech/coordinator/client"

	bridge_config "scroll-tech/bridge/config"

	coordinator_config "scroll-tech/coordinator/config"
)

const managerURL = "localhost:8132"
const managerPort = ":8132"

var (
	cfg              *bridge_config.Config
	l2gethImg, dbImg docker.ImgInstance
	rollerManager    *coordinator.Manager
	handle           *http.Server
)

func setEnv(t *testing.T) error {
	var err error
	// Load config.
	cfg, err = bridge_config.NewConfig("../bridge/config.json")
	assert.NoError(t, err)

	// Create l2geth container.
	l2gethImg = docker.NewTestL2Docker(t)
	cfg.L1Config.RelayerConfig.SenderConfig.Endpoint = l2gethImg.Endpoint()
	cfg.L2Config.Endpoint = l2gethImg.Endpoint()

	// Create db container.
	dbImg = docker.NewTestDBDocker(t, cfg.DBConfig.DriverName)
	cfg.DBConfig.DSN = dbImg.Endpoint()

	// start roller manager
	rollerManager = setupRollerManager(t, "", cfg.DBConfig)

	// start ws service
	handle, _, err = utils.StartWSEndpoint(managerURL, rollerManager.APIs())
	assert.NoError(t, err)
	return err
}

func TestApis(t *testing.T) {
	// Set up the test environment.
	assert.True(t, assert.NoError(t, setEnv(t)), "failed to setup the test environment.")

	t.Run("TestHandshake", testHandshake)
	t.Run("TestSeveralConnections", testSeveralConnections)
	t.Run("TestIdleRollerSelection", testIdleRollerSelection)

	// Teardown
	t.Cleanup(func() {
		handle.Shutdown(context.Background())
		rollerManager.Stop()
		l2gethImg.Stop()
		dbImg.Stop()
	})
}

func testHandshake(t *testing.T) {
	// Create db handler and reset db.
	l2db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(l2db.GetDB().DB))
	defer l2db.Close()

	stopCh := make(chan struct{})
	performHandshake(t, 1, "roller_test", "ws://"+managerURL, stopCh)

	assert.Equal(t, 1, rollerManager.GetNumberOfIdleRollers())

	close(stopCh)
}

func testSeveralConnections(t *testing.T) {
	// Create db handler and reset db.
	l2db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(l2db.GetDB().DB))
	defer l2db.Close()

	var (
		batch  = 100
		stopCh = make(chan struct{})
		eg     = errgroup.Group{}
	)
	for i := 0; i < batch; i++ {
		idx := i
		eg.Go(func() error {
			performHandshake(t, 1, "roller_test"+strconv.Itoa(idx), "ws://"+managerURL, stopCh)
			return nil
		})
	}
	assert.NoError(t, eg.Wait())

	// check roller's idle connections
	assert.Equal(t, batch, rollerManager.GetNumberOfIdleRollers())

	// close connection
	close(stopCh)

	var (
		tick     = time.Tick(time.Second)
		tickStop = time.Tick(time.Second * 10)
	)
	for {
		select {
		case <-tick:
			if rollerManager.GetNumberOfIdleRollers() == 0 {
				return
			}
		case <-tickStop:
			t.Error("roller connect is blocked")
			return
		}
	}
}

func testIdleRollerSelection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create db handler and reset db.
	l2db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(l2db.GetDB().DB))
	defer l2db.Close()

	var (
		l2cfg = cfg.L2Config
		l2Cli *ethclient.Client
	)
	l2Cli, err = ethclient.Dial(l2cfg.Endpoint)
	assert.NoError(t, err)
	rc := l2.NewL2WatcherClient(context.Background(), l2Cli, l2cfg.Confirmations, l2cfg.ProofGenerationFreq, l2cfg.SkippedOpcodes, l2cfg.L2MessengerAddress, l2db)
	rc.Start()
	defer rc.Stop()

	// create ws connections.
	batch := 20
	stopCh := make(chan struct{})
	for i := 0; i < batch; i++ {
		performHandshake(t, 1, "roller_test"+strconv.Itoa(i), "ws://"+managerURL, stopCh)
	}
	assert.Equal(t, batch, rollerManager.GetNumberOfIdleRollers())

	relayCfg := cfg.L1Config.RelayerConfig
	relayCfg.SenderConfig.Confirmations = 0
	to := common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
	newSender, err := sender.NewSender(ctx, relayCfg.SenderConfig, relayCfg.MessageSenderPrivateKeys)
	assert.True(t, assert.NoError(t, err), "unable to create a sender instance.")
	for i := 0; i < 2; i++ {
		_, err = newSender.SendTransaction(strconv.Itoa(1000+i), &to, big.NewInt(1000000000), nil)
		assert.NoError(t, err)
		<-newSender.ConfirmChan()
	}

	// Get the latest block number.
	latest, err := l2Cli.BlockNumber(ctx)
	assert.NoError(t, err)

	// verify proof status
	var (
		number   int64
		tick     = time.Tick(time.Second)
		tickStop = time.Tick(10 * 60 * time.Second)
	)
	for {
		select {
		case <-tick:
			// get the latest number
			if number, err = l2db.GetBlockTracesLatestHeight(); err != nil || number < int64(latest) {
				continue
			}
			infos, err := l2db.GetBlockInfos(map[string]interface{}{"number": number}, "LIMIT 1")
			if err != nil || len(infos) == 0 || !infos[0].BatchID.Valid {
				continue
			}
			batchID := infos[0].BatchID.String
			status, err := l2db.GetProvingStatusByID(batchID)
			if err == nil && status == orm.ProvingTaskVerified {
				return
			}
		case <-tickStop:
			t.Error("failed to check proof status")
			close(stopCh)
			return
		}
	}
}

func setupRollerManager(t *testing.T, verifierEndpoint string, dbCfg *database.DBConfig) *coordinator.Manager {
	// Get db handler.
	db, err := database.NewOrmFactory(dbCfg)
	assert.True(t, assert.NoError(t, err), "failed to get db handler.")
	// Reset db.
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB), "failed to reset db.")

	rollerManager, err := coordinator.New(context.Background(), &coordinator_config.RollerManagerConfig{
		Endpoint:          managerPort,
		RollersPerSession: 1,
		VerifierEndpoint:  verifierEndpoint,
		CollectionTime:    1,
	}, db)
	assert.NoError(t, err)
	assert.NoError(t, rollerManager.Start())

	return rollerManager
}

// performHandshake sets up a websocket client to connect to the roller manager.
func performHandshake(t *testing.T, proofTime time.Duration, name string, wsURL string, stopCh chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create a new ws connection
	client, err := client2.DialContext(ctx, wsURL)
	assert.NoError(t, err)

	// create private key
	privkey, err := crypto.GenerateKey()
	assert.NoError(t, err)

	authMsg := &message.AuthMessage{
		Identity: &message.Identity{
			Name:      name,
			Timestamp: time.Now().UnixNano(),
		},
	}
	assert.NoError(t, authMsg.Sign(privkey))

	traceCh := make(chan *message.TaskMsg, 4)
	sub, err := client.RegisterAndSubscribe(ctx, traceCh, authMsg)
	if err != nil {
		t.Error(err)
		return
	}

	go func() {
		for {
			select {
			case trace := <-traceCh:
				id := trace.ID
				// sleep several seconds to mock the proof process.
				<-time.After(proofTime * time.Second)
				proof := &message.AuthZkProof{
					ProofMsg: &message.ProofMsg{
						ID:     id,
						Status: message.StatusOk,
						Proof:  &message.AggProof{},
					},
				}
				assert.NoError(t, proof.Sign(privkey))
				ok, err := client.SubmitProof(context.Background(), proof)
				if err != nil {
					t.Error(err)
				}
				assert.Equal(t, true, ok)
			case <-stopCh:
				sub.Unsubscribe()
				return
			}
		}
	}()
}
