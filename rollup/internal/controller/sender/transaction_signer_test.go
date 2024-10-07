package sender

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/holiman/uint256"
	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/testcontainers"

	"scroll-tech/rollup/internal/config"
)

var (
	testAppsSignerTest *testcontainers.TestcontainerApps
	chainId            int
)

func setupEnvSignerTest(t *testing.T) {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.LogfmtFormat()))
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)

	chainId = 1
	testAppsSignerTest = testcontainers.NewTestcontainerApps()
	assert.NoError(t, testAppsSignerTest.StartWeb3SignerContainer(chainId))
}

func TestTransactionSigner(t *testing.T) {
	setupEnvSignerTest(t)
	t.Run("test both signer types", testBothSignerTypes)
}

func testBothSignerTypes(t *testing.T) {
	endpoint, err := testAppsSignerTest.GetWeb3SignerEndpoint()
	assert.NoError(t, err)

	// create remote signer
	remoteSignerConf := &config.SignerConfig{
		SignerType: RemoteSignerType,
		RemoteSignerConfig: &config.RemoteSignerConfig{
			SignerAddress:   "0x1C5A77d9FA7eF466951B2F01F724BCa3A5820b63",
			RemoteSignerUrl: endpoint,
		},
	}
	remoteSigner, err := NewTransactionSigner(remoteSignerConf, big.NewInt(int64(chainId)))
	assert.NoError(t, err)
	remoteSigner.SetNonce(2)

	// create private key signer
	privateKeySignerConf := &config.SignerConfig{
		SignerType: PrivateKeySignerType,
		PrivateKeySignerConfig: &config.PrivateKeySignerConfig{
			PrivateKey: "1212121212121212121212121212121212121212121212121212121212121212",
		},
	}
	privateKeySigner, err := NewTransactionSigner(privateKeySignerConf, big.NewInt(int64(chainId)))
	assert.NoError(t, err)
	privateKeySigner.SetNonce(2)

	assert.Equal(t, remoteSigner.GetAddr(), privateKeySigner.GetAddr())

	to := common.BytesToAddress([]byte{0, 1, 2, 3})
	data := []byte("data")

	// check LegacyTx and DynamicFeeTx - transactions supported by web3signer
	txDatas := []gethTypes.TxData{
		&gethTypes.LegacyTx{
			Nonce:    remoteSigner.GetNonce(),
			GasPrice: big.NewInt(1000),
			Gas:      10000,
			To:       &to,
			Data:     data,
		},
		&gethTypes.DynamicFeeTx{
			Nonce:     remoteSigner.GetNonce(),
			Gas:       10000,
			To:        &to,
			Data:      data,
			ChainID:   big.NewInt(int64(chainId)),
			GasTipCap: big.NewInt(2000),
			GasFeeCap: big.NewInt(3000),
		},
	}
	var signedTx1 *gethTypes.Transaction
	var signedTx2 *gethTypes.Transaction
	for _, txData := range txDatas {
		tx := gethTypes.NewTx(txData)

		signedTx1, err = remoteSigner.SignTransaction(context.Background(), tx)
		assert.NoError(t, err)

		signedTx2, err = privateKeySigner.SignTransaction(context.Background(), tx)
		assert.NoError(t, err)

		assert.Equal(t, signedTx1.Hash(), signedTx2.Hash())
	}

	// BlobTx is not supported
	txData := &gethTypes.BlobTx{
		Nonce:      remoteSigner.GetNonce(),
		Gas:        10000,
		To:         to,
		Data:       data,
		ChainID:    uint256.NewInt(1),
		GasTipCap:  uint256.NewInt(2000),
		GasFeeCap:  uint256.NewInt(3000),
		BlobFeeCap: uint256.NewInt(1),
		BlobHashes: []common.Hash{},
		Sidecar:    nil,
	}
	tx := gethTypes.NewTx(txData)

	_, err = remoteSigner.SignTransaction(context.Background(), tx)
	assert.Error(t, err)
}
