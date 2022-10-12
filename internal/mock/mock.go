package mock

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"io"
	"math/big"
	"net"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/gorilla/websocket"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

	rollers "scroll-tech/coordinator"
	coordinator_config "scroll-tech/coordinator/config"
	"scroll-tech/coordinator/message"
	"scroll-tech/store"
	db_config "scroll-tech/store/config"
	"scroll-tech/store/migrate"

	"scroll-tech/bridge"
	bridge_config "scroll-tech/bridge/config"
	"scroll-tech/bridge/l2"

	"scroll-tech/internal/docker"
)

// PerformHandshake sets up a websocket client to connect to the roller manager.
func PerformHandshake(assert *assert.Assertions, c *websocket.Conn) {
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
	assert.NoError(err)
	sig, err := secp256k1.Sign(hash, sk)
	assert.NoError(err)

	authMsg.Signature = common.Bytes2Hex(sig)

	b, err := json.Marshal(authMsg)
	assert.NoError(err)

	msg := &message.Msg{
		Type:    message.Register,
		Payload: b,
	}

	b, err = json.Marshal(msg)
	assert.NoError(err)

	assert.NoError(c.WriteMessage(websocket.BinaryMessage, b))
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

// SetupMockVerifier sets up a mocked verifier for a test case.
func SetupMockVerifier(assert *assert.Assertions, verifierEndpoint string) {
	err := os.RemoveAll(verifierEndpoint)
	assert.NoError(err)

	l, err := net.Listen("unix", verifierEndpoint)
	assert.NoError(err)

	go func() {
		conn, err := l.Accept()
		assert.NoError(err)

		// Simply read all incoming messages and send a true boolean straight back.
		for {
			// Read length
			buf := make([]byte, 4)
			_, err = io.ReadFull(conn, buf)
			assert.NoError(err)

			// Read message
			msgLength := binary.LittleEndian.Uint64(buf)
			buf = make([]byte, msgLength)
			_, err = io.ReadFull(conn, buf)
			assert.NoError(err)

			// Return boolean
			buf = []byte{1}
			_, err = conn.Write(buf)
			assert.NoError(err)
		}
	}()

}

type L1GethTestConfig struct {
	HPort int
	WPort int
}

type L2GethTestConfig struct {
	HPort int
	WPort int
}

type DbTestconfig struct {
	DbName    string
	DbPort    int
	DB_CONFIG *db_config.DBConfig
}

type TestConfig struct {
	L1GethTestConfig
	L2GethTestConfig
	DbTestconfig
}

func NewTestL1Docker(t *testing.T, tcfg *TestConfig) docker.ImgInstance {
	img_geth := docker.NewImgGeth(t, "scroll_l1geth", "", "", tcfg.L1GethTestConfig.HPort, tcfg.L1GethTestConfig.WPort)
	assert.NoError(t, img_geth.Start())
	return img_geth
}

func GetTestL2Docker(t *testing.T, tcfg *TestConfig) docker.ImgInstance {
	img_geth := docker.NewImgGeth(t, "scroll_l2geth", "", "", tcfg.L2GethTestConfig.HPort, tcfg.L2GethTestConfig.WPort)
	assert.NoError(t, img_geth.Start())
	return img_geth
}

func GetDbDocker(t *testing.T, tcfg *TestConfig) docker.ImgInstance {
	img_db := docker.NewImgDB(t, "postgres", "123456", tcfg.DbName, tcfg.DbPort)
	assert.NoError(t, img_db.Start())
	return img_db
}

// Mockl2geth return mock l2geth client created with docker for test
func Mockl2gethDocker(t *testing.T, cfg *bridge_config.Config, tcfg *TestConfig) (bridge.MockL2BackendClient, docker.ImgInstance, docker.ImgInstance) {
	// initialize l2geth docker image
	img_geth := GetTestL2Docker(t, tcfg)

	cfg.L2Config.Endpoint = img_geth.Endpoint()

	// initialize db docker image
	img_db := GetDbDocker(t, tcfg)

	db, err := store.NewOrmFactory(&db_config.DBConfig{
		DriverName: "postgres",
		DSN:        img_db.Endpoint(),
	})
	assert.NoError(t, err)

	client, err := l2.New(context.Background(), cfg.L2Config, db)
	assert.NoError(t, err)

	return client, img_geth, img_db
}

// SetupRollerManager return rollers.Manager for testcase
func SetupRollerManager(assert *assert.Assertions, cfg *coordinator_config.Config, orm store.OrmFactory) *rollers.Manager {
	// Load config file.
	ctx := context.Background()

	SetupMockVerifier(assert, cfg.RollerManagerConfig.VerifierEndpoint)
	rollerManager, err := rollers.New(ctx, cfg.RollerManagerConfig, orm)
	assert.NoError(err)

	// Start rollermanager modules.
	err = rollerManager.Start()
	assert.NoError(err)
	return rollerManager
}

// MockClearDB clears db
func MockClearDB(assert *assert.Assertions, db_cfg *db_config.DBConfig) {
	db, err := store.NewConnection(db_cfg)
	assert.NoError(err)
	version0 := int64(0)
	err = migrate.Rollback(db.DB, &version0)
	assert.NoError(err)
	err = migrate.Migrate(db.DB)
	assert.NoError(err)
	err = db.DB.Close()
	assert.NoError(err)
}

// MockPrepareDB will return DB for testcase
func MockPrepareDB(assert *assert.Assertions, db_cfg *db_config.DBConfig) store.OrmFactory {
	db, err := store.NewOrmFactory(db_cfg)
	assert.NoError(err)
	return db
}

// MockSendTxToL2Client will send a default Tx by calling l2geth client
func MockSendTxToL2Client(assert *assert.Assertions, client *ethclient.Client) {
	privateKey, err := crypto.HexToECDSA("ad29c7c341a23f04851b6c8602c7c74b98e3fc9488582791bda60e0e261f9cbb")
	assert.NoError(err)
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	assert.True(ok)
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	assert.NoError(err)
	value := big.NewInt(1000000000) // in wei
	gasLimit := uint64(30000000)    // in units
	gasPrice, err := client.SuggestGasPrice(context.Background())
	assert.NoError(err)
	toAddress := common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
	tx := types.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, nil)
	chainID, err := client.ChainID(context.Background())
	assert.NoError(err)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	assert.NoError(err)

	assert.NoError(client.SendTransaction(context.Background(), signedTx))
}
