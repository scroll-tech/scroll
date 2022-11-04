package coordinator_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/gorilla/websocket"
	"net/http"
	"scroll-tech/common/message"
	"scroll-tech/database/migrate"
	"strconv"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/ethclient"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"

	"scroll-tech/common/docker"
	"scroll-tech/common/utils"
	"scroll-tech/coordinator"
	"scroll-tech/database"
	"scroll-tech/database/orm"

	client2 "scroll-tech/coordinator/client"

	bridge_config "scroll-tech/bridge/config"

	coordinator_config "scroll-tech/coordinator/config"
)

const managerURL = "localhost:8132"
const managerPort = ":8132"

var (
	cfg              *bridge_config.Config
	l2gethImg, dbImg docker.ImgInstance
	db               database.OrmFactory
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

	// Create db handler and reset db.
	db, err = database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))

	// start roller manager
	rollerManager = setupRollerManager(t, "", db)

	// start ws service
	handle, _, err = utils.StartWSEndpoint(managerURL, rollerManager.APIs())
	assert.NoError(t, err)
	return err
}

func TestApis(t *testing.T) {
	assert.True(t, assert.NoError(t, setEnv(t)))

	t.Run("TestHandshake", testHandshake)
	t.Run("TestSeveralConnections", testSeveralConnections)
	t.Run("TestIdleRollerSelection", testIdleRollerSelection)

	t.Cleanup(func() {
		handle.Shutdown(context.Background())
		rollerManager.Stop()
		l2gethImg.Stop()
		db.Close()
		dbImg.Stop()
	})
}

func testHandshake(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// reset db
	mock.ResetDB(t, TEST_CONFIG.DB_CONFIG)

	// create a new
	client, err := client2.DialContext(ctx, "ws://"+managerURL)
	assert.NoError(t, err)

	stopCh := make(chan struct{})
	mock.PerformHandshake(t, 1, "roller_test", client, stopCh)

	assert.Equal(t, 1, rollerManager.GetNumberOfIdleRollers())

	close(stopCh)
}

func testSeveralConnections(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		batch  = 100
		stopCh = make(chan struct{})
		eg     = errgroup.Group{}
	)
	for i := 0; i < batch; i++ {
		idx := i
		eg.Go(func() error {
			// create a new ws connection
			client, err := client2.DialContext(ctx, "ws://"+managerURL)
			assert.NoError(t, err)
			mock.PerformHandshake(t, 1, "roller_test"+strconv.Itoa(idx), client, stopCh)
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

	// reset db
	traceDB := orm.BlockResultOrm(mock.ResetDB(t, TEST_CONFIG.DB_CONFIG))

	// create l2geth client
	ethClient, err := ethclient.DialContext(ctx, l2gethImg.Endpoint())
	assert.NoError(t, err)

	// create ws connections.
	batch := 20
	stopCh := make(chan struct{})
	for i := 0; i < batch; i++ {
		var client *client2.Client
		client, err = client2.DialContext(ctx, "ws://"+managerURL)
		assert.NoError(t, err)
		mock.PerformHandshake(t, 1, "roller_test"+strconv.Itoa(i), client, stopCh)
	}
	assert.Equal(t, batch, rollerManager.GetNumberOfIdleRollers())

	// send two txs
	mock.SendTxToL2Client(t, ethClient, cfg.L2Config.RelayerConfig.PrivateKey)
	mock.SendTxToL2Client(t, ethClient, cfg.L2Config.RelayerConfig.PrivateKey)

	// verify proof status
	var (
		number   int64 = 1
		latest   int64
		tick     = time.Tick(time.Second)
		tickStop = time.Tick(time.Second * 20)
	)
	for {
		select {
		case <-tick:
			// get the latest number
			if latest, err = traceDB.GetBlockResultsLatestHeight(); err != nil || latest <= 0 {
				continue
			}
			if number > latest {
				close(stopCh)
				return
			}
			status, err := traceDB.GetBlockStatusByNumber(uint64(number))
			if err == nil && (status == orm.BlockVerified || status == orm.BlockSkipped) {
				number++
			}
		case <-tickStop:
			t.Error("failed to check proof status")
			close(stopCh)
			return
		}
	}
}

func setupRollerManager(t *testing.T, verifierEndpoint string, orm database.OrmFactory) *coordinator.Manager {
	rollerManager, err := coordinator.New(context.Background(), &coordinator_config.RollerManagerConfig{
		Endpoint:          managerPort,
		RollersPerSession: 1,
		VerifierEndpoint:  verifierEndpoint,
		CollectionTime:    1,
	}, orm)
	assert.NoError(t, err)

	assert.NoError(t, rollerManager.Start())

	return rollerManager
}

// performHandshake sets up a websocket client to connect to the roller manager.
func performHandshake(t *testing.T, c *websocket.Conn) {
	// Try to perform handshake
	pk, sk := generateKeyPair()
	authMsg := &message.AuthMessage{
		Identity: message.Identity{
			Name:      "testRoller",
			Timestamp: time.Now().UnixNano(),
			PublicKey: common.Bytes2Hex(pk),
		},
		Signature: "",
	}

	hash, err := authMsg.Identity.Hash()
	assert.NoError(t, err)
	sig, err := secp256k1.Sign(hash, sk)
	assert.NoError(t, err)

	authMsg.Signature = common.Bytes2Hex(sig)

	b, err := json.Marshal(authMsg)
	assert.NoError(t, err)

	msg := &message.Msg{
		Type:    message.Register,
		Payload: b,
	}

	b, err = json.Marshal(msg)
	assert.NoError(t, err)

	assert.NoError(t, c.WriteMessage(websocket.BinaryMessage, b))
}

func generateKeyPair() (pubkey, privkey []byte) {
	key, err := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	pubkey = elliptic.Marshal(secp256k1.S256(), key.X, key.Y)

	privkey = make([]byte, 32)
	blob := key.D.Bytes()
	copy(privkey[32-len(blob):], blob)

	return pubkey, privkey
}
