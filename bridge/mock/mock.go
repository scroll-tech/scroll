package mock

import (
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"io"
	"math/big"
	"net"
	"os"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/docker"
	"scroll-tech/common/message"
	"scroll-tech/coordinator"
	coordinator_config "scroll-tech/coordinator/config"
	"scroll-tech/database"
	db_docker "scroll-tech/database/docker"
	"scroll-tech/database/migrate"

	db_config "scroll-tech/database"

	client2 "scroll-tech/coordinator/client"

	bridge_config "scroll-tech/bridge/config"
	"scroll-tech/bridge/l2"
)

// PerformHandshake sets up a websocket client to connect to the roller manager.
func PerformHandshake(t *testing.T, proofTime time.Duration, name string, client *client2.Client, stopCh chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
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

	traceCh := make(chan *message.BlockTraces, 4)
	sub, err := client.RegisterAndSubscribe(ctx, traceCh, authMsg)
	if err != nil {
		t.Error(err)
		return
	}

	go func() {
		for {
			select {
			case trace := <-traceCh:
				id := trace.Traces.BlockTrace.Number.ToInt().Uint64()
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

// SetupMockVerifier sets up a mocked verifier for a test case.
func SetupMockVerifier(t *testing.T, verifierEndpoint string) {
	err := os.RemoveAll(verifierEndpoint)
	assert.NoError(t, err)

	l, err := net.Listen("unix", verifierEndpoint)
	assert.NoError(t, err)

	go func() {
		conn, err := l.Accept()
		assert.NoError(t, err)

		// Simply read all incoming messages and send a true boolean straight back.
		for {
			// Read length
			buf := make([]byte, 4)
			_, err = io.ReadFull(conn, buf)
			assert.NoError(t, err)

			// Read message
			msgLength := binary.LittleEndian.Uint64(buf)
			buf = make([]byte, msgLength)
			_, err = io.ReadFull(conn, buf)
			assert.NoError(t, err)

			// Return boolean
			buf = []byte{1}
			_, err = conn.Write(buf)
			assert.NoError(t, err)
		}
	}()

}

// L1GethTestConfig is the http and web socket port of l1geth docker
type L1GethTestConfig struct {
	HPort int
	WPort int
}

// L2GethTestConfig is the http and web socket port of l2geth docker
type L2GethTestConfig struct {
	HPort int
	WPort int
}

// DbTestconfig is the test config of database docker
type DbTestconfig struct {
	DbName    string
	DbPort    int
	DB_CONFIG *database.DBConfig
}

// TestConfig is the config for test
type TestConfig struct {
	L1GethTestConfig
	L2GethTestConfig
	DbTestconfig
}

// NewL1Docker starts and returns l1geth docker
func NewL1Docker(t *testing.T, tcfg *TestConfig) docker.ImgInstance {
	imgGeth := docker.NewImgGeth(t, "scroll_l1geth", "", "", tcfg.L1GethTestConfig.HPort, tcfg.L1GethTestConfig.WPort)
	assert.NoError(t, imgGeth.Start())
	return imgGeth
}

// NewL2Docker starts and returns l2geth docker
func NewL2Docker(t *testing.T, tcfg *TestConfig) docker.ImgInstance {
	imgGeth := docker.NewImgGeth(t, "scroll_l2geth", "", "", tcfg.L2GethTestConfig.HPort, tcfg.L2GethTestConfig.WPort)
	assert.NoError(t, imgGeth.Start())
	return imgGeth
}

// NewDBDocker starts and returns database docker
func NewDBDocker(t *testing.T, tcfg *TestConfig) docker.ImgInstance {
	imgDb := db_docker.NewImgDB(t, "postgres", "123456", tcfg.DbName, tcfg.DbPort)
	assert.NoError(t, imgDb.Start())
	return imgDb
}

// L2gethDocker return mock l2geth client created with docker for test
func L2gethDocker(t *testing.T, cfg *bridge_config.Config, tcfg *TestConfig) (*l2.Backend, docker.ImgInstance, docker.ImgInstance) {
	// initialize l2geth docker image
	imgGeth := NewL2Docker(t, tcfg)

	cfg.L2Config.Endpoint = imgGeth.Endpoint()

	// initialize db docker image
	imgDb := NewDBDocker(t, tcfg)

	db := PrepareDB(t, tcfg.DB_CONFIG)
	assert.Equal(t, true, db != nil)

	client, err := l2.New(context.Background(), cfg.L2Config, db)
	assert.NoError(t, err)
	assert.NoError(t, client.Start())

	return client, imgGeth, imgDb
}

// SetupRollerManager return rollers.Manager for testcase
func SetupRollerManager(t *testing.T, cfg *coordinator_config.RollerManagerConfig, orm database.OrmFactory) *coordinator.Manager {
	// Load config file.
	ctx := context.Background()

	if cfg.VerifierEndpoint != "" {
		SetupMockVerifier(t, cfg.VerifierEndpoint)
	}
	rollerManager, err := coordinator.New(ctx, cfg, orm)
	assert.NoError(t, err)

	// Start rollermanager modules.
	err = rollerManager.Start()
	assert.NoError(t, err)
	return rollerManager
}

// PrepareDB clears and reset db
func PrepareDB(t *testing.T, db_cfg *db_config.DBConfig) database.OrmFactory {
	db, err := database.NewOrmFactory(db_cfg)
	assert.NoError(t, err)

	version0 := int64(0)
	assert.NoError(t, migrate.Rollback(db.GetDB().DB, &version0))
	assert.NoError(t, migrate.Migrate(db.GetDB().DB))
	return db
}

// SendTxToL2Client will send a default Tx by calling l2geth client
func SendTxToL2Client(t *testing.T, client *ethclient.Client, private string) {
	privateKey, err := crypto.HexToECDSA(private)
	assert.NoError(t, err)
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	assert.True(t, ok)
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	assert.NoError(t, err)
	value := big.NewInt(1000000000) // in wei
	gasLimit := uint64(30000000)    // in units
	gasPrice, err := client.SuggestGasPrice(context.Background())
	assert.NoError(t, err)
	toAddress := common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
	tx := types.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, nil)
	chainID, err := client.ChainID(context.Background())
	assert.NoError(t, err)

	// sign tx
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	assert.NoError(t, err)

	// send tx
	assert.NoError(t, client.SendTransaction(context.Background(), signedTx))

	// wait util on chain
	_, err = bind.WaitMined(context.Background(), client, signedTx)
	assert.NoError(t, err)
}
