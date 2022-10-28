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

	"scroll-tech/database"
	"scroll-tech/database/migrate"

	"scroll-tech/common/message"
	"scroll-tech/coordinator"
	coordinator_config "scroll-tech/coordinator/config"

	bridge_config "scroll-tech/bridge/config"
	"scroll-tech/bridge/l2"

	"scroll-tech/common/docker"

	docker_db "scroll-tech/database/docker"
)

// PerformHandshake sets up a websocket client to connect to the roller manager.
func PerformHandshake(t *testing.T, c *websocket.Conn) {
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

// NewTestL1Docker starts and returns l1geth docker
func NewTestL1Docker(t *testing.T, tcfg *TestConfig) docker.ImgInstance {
	img_geth := docker.NewImgGeth(t, "scroll_l1geth", "", "", tcfg.L1GethTestConfig.HPort, tcfg.L1GethTestConfig.WPort)
	assert.NoError(t, img_geth.Start())
	return img_geth
}

// NewTestL2Docker starts and returns l2geth docker
func NewTestL2Docker(t *testing.T, tcfg *TestConfig) docker.ImgInstance {
	img_geth := docker.NewImgGeth(t, "scroll_l2geth", "", "", tcfg.L2GethTestConfig.HPort, tcfg.L2GethTestConfig.WPort)
	assert.NoError(t, img_geth.Start())
	return img_geth
}

// GetDbDocker starts and returns database docker
func GetDbDocker(t *testing.T, tcfg *TestConfig) docker.ImgInstance {
	img_db := docker_db.NewImgDB(t, "postgres", "123456", tcfg.DbName, tcfg.DbPort)
	assert.NoError(t, img_db.Start())
	return img_db
}

// L2gethDocker return mock l2geth client created with docker for test
func L2gethDocker(t *testing.T, cfg *bridge_config.Config, tcfg *TestConfig) (*l2.Backend, docker.ImgInstance, docker.ImgInstance) {
	// initialize l2geth docker image
	img_geth := NewTestL2Docker(t, tcfg)

	cfg.L2Config.Endpoint = img_geth.Endpoint()

	// initialize db docker image
	img_db := GetDbDocker(t, tcfg)

	db, err := database.NewOrmFactory(&database.DBConfig{
		DriverName: "postgres",
		DSN:        img_db.Endpoint(),
	})
	assert.NoError(t, err)

	client, err := l2.New(context.Background(), cfg.L2Config, db)
	assert.NoError(t, err)

	return client, img_geth, img_db
}

// SetupRollerManager return coordinator.Manager for testcase
func SetupRollerManager(t *testing.T, cfg *coordinator_config.Config, orm database.OrmFactory) *coordinator.Manager {
	// Load config file.
	ctx := context.Background()

	SetupMockVerifier(t, cfg.RollerManagerConfig.VerifierEndpoint)
	rollerManager, err := coordinator.New(ctx, cfg.RollerManagerConfig, orm)
	assert.NoError(t, err)

	// Start rollermanager modules.
	err = rollerManager.Start()
	assert.NoError(t, err)
	return rollerManager
}

// ClearDB clears db
func ClearDB(t *testing.T, db_cfg *database.DBConfig) {
	factory, err := database.NewOrmFactory(db_cfg)
	assert.NoError(t, err)
	db := factory.GetDB()
	version0 := int64(0)
	err = migrate.Rollback(db.DB, &version0)
	assert.NoError(t, err)
	err = migrate.Migrate(db.DB)
	assert.NoError(t, err)
	err = db.DB.Close()
	assert.NoError(t, err)
}

// PrepareDB will return DB for testcase
func PrepareDB(t *testing.T, db_cfg *database.DBConfig) database.OrmFactory {
	db, err := database.NewOrmFactory(db_cfg)
	assert.NoError(t, err)
	return db
}

// SendTxToL2Client will send a default Tx by calling l2geth client
func SendTxToL2Client(t *testing.T, client *ethclient.Client, privateKey *ecdsa.PrivateKey) *types.Transaction {
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

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	assert.NoError(t, err)

	assert.NoError(t, client.SendTransaction(context.Background(), signedTx))
	return signedTx
}
