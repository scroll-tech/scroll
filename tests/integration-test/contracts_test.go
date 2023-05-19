package integration_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"scroll-tech/database"
	"scroll-tech/database/migrate"
	"scroll-tech/database/orm"

	"scroll-tech/common/bytecode/erc20"
	"scroll-tech/common/bytecode/greeter"
	l1gateway "scroll-tech/common/bytecode/scroll/L1/ethgateway"
	"scroll-tech/common/bytecode/scroll/L1/scrollchain"
	l2gateway "scroll-tech/common/bytecode/scroll/L2/ethgateway"
	ctypes "scroll-tech/common/types"
	"scroll-tech/common/utils"
)

var (
	// Balance equal to 1e28
	amount = new(big.Int).SetBytes(common.FromHex("204fce5e3e25020000000000"))
	ether  = big.NewInt(1e18)
)

func TestERC20(t *testing.T) {
	base.RunL2Geth(t)
	l2Cli, err := base.L2Client()
	assert.Nil(t, err)

	token, err := erc20.NewERC20Mock(base.ERC20, l2Cli)
	assert.NoError(t, err)

	auth, err := bind.NewKeyedTransactorWithChainID(bridgeApp.Config.L2Config.RelayerConfig.GasOracleSenderPrivateKeys[0], base.L2gethImg.ChainID())
	assert.NoError(t, err)

	authBls0, err := token.BalanceOf(nil, auth.From)
	assert.NoError(t, err)

	tokenBls0, err := token.BalanceOf(nil, base.ERC20)
	assert.NoError(t, err)

	// create tx to transfer balance.
	value := big.NewInt(1000)
	to := common.HexToAddress("0x85fd9d96a42972f8301b886e77838f363e72dff7")
	tx, err := token.Transfer(auth, to, value)
	assert.NoError(t, err)
	_, err = bind.WaitMined(context.Background(), l2Cli, tx)
	assert.NoError(t, err)

	// check balance.
	authBls1, err := token.BalanceOf(nil, auth.From)
	assert.NoError(t, err)
	tokenBls1, err := token.BalanceOf(nil, to)
	assert.NoError(t, err)
	assert.Equal(t, authBls0.Int64(), authBls1.Add(authBls1, value).Int64())
	assert.Equal(t, tokenBls1.Int64(), tokenBls0.Add(tokenBls0, value).Int64())
}

func TestGreeter(t *testing.T) {
	base.RunL2Geth(t)
	l2Cli, err := base.L2Client()
	assert.Nil(t, err)

	chainID, _ := l2Cli.ChainID(context.Background())
	auth, err := bind.NewKeyedTransactorWithChainID(bridgeApp.Config.L2Config.RelayerConfig.GasOracleSenderPrivateKeys[0], chainID)
	assert.NoError(t, err)

	token, err := greeter.NewGreeter(base.Greeter, l2Cli)
	assert.NoError(t, err)

	val := big.NewInt(100)
	tx, err := token.SetValue(auth, val)
	assert.NoError(t, err)
	_, err = bind.WaitMined(context.Background(), l2Cli, tx)
	assert.NoError(t, err)

	// check result.
	res, err := token.Retrieve(nil)
	assert.NoError(t, err)
	assert.Equal(t, val.String(), res.String())
}

func TestDepositAndWithdraw(t *testing.T) {
	base.RunImages(t)
	assert.NoError(t, migrate.ResetDB(base.DBClient(t)))

	// Run bridge apps.
	bridgeApp.RunApp(t, utils.EventWatcherApp)
	bridgeApp.RunApp(t, utils.GasOracleApp)
	bridgeApp.RunApp(t, utils.MessageRelayerApp)
	bridgeApp.RunApp(t, utils.RollupRelayerApp)

	// Run coordinator app.
	coordinatorApp.RunApp(t)
	// Run roller app.
	rollerApp.RunApp(t)

	t.Run("TestGenesisBatch", testGenesisBatch)
	t.Run("TestETHDeposit", testETHDeposit)
	t.Run("TestETHWithdraw", testETHWithdraw)

	// Free apps.
	bridgeApp.WaitExit()
	rollerApp.WaitExit()
	coordinatorApp.WaitExit()
}

func testGenesisBatch(t *testing.T) {
	l1Cli, err := base.L1Client()
	assert.NoError(t, err)
	l2Cli, err := base.L2Client()
	assert.NoError(t, err)

	scrollChain, err := scrollchain.NewScrollChain(base.L1Contracts.L1ScrollChain, l1Cli)
	assert.NoError(t, err)

	// Create genesis batch.
	genesis, err := l2Cli.HeaderByNumber(context.Background(), big.NewInt(0))
	assert.NoError(t, err)
	batchData0 := ctypes.NewGenesisBatchData(&ctypes.WrappedBlock{Header: genesis, WithdrawTrieRoot: common.Hash{}})

	// Check genesis batch is imported or not.
	expectBatch, err := scrollChain.Batches(nil, *batchData0.Hash())
	assert.NoError(t, err)
	if expectBatch.NewStateRoot == batchData0.Batch.NewStateRoot {
		return
	}

	// Import genesis batch.
	l1ChainID, _ := l1Cli.ChainID(context.Background())
	l1Auth, err := bind.NewKeyedTransactorWithChainID(bridgeApp.Config.L2Config.RelayerConfig.GasOracleSenderPrivateKeys[0], l1ChainID)
	assert.NoError(t, err)
	tx, err := scrollChain.ImportGenesisBatch(l1Auth, translateBatch(batchData0))
	assert.NoError(t, err)
	receipt, err := bind.WaitMined(context.Background(), l1Cli, tx)
	assert.NoError(t, err)
	assert.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Make sure genesis batch is exist.
	expectBatch, err = scrollChain.Batches(nil, *batchData0.Hash())
	assert.NoError(t, err)
	assert.Equal(t, batchData0.Batch.NewStateRoot, common.BytesToHash(expectBatch.NewStateRoot[:]))
}

func testETHDeposit(t *testing.T) {
	l1Cli, err := base.L1Client()
	assert.NoError(t, err)
	l2Cli, err := base.L2Client()
	assert.NoError(t, err)

	l1ChainID, _ := l1Cli.ChainID(context.Background())
	l1Auth, err := bind.NewKeyedTransactorWithChainID(bridgeApp.Config.L2Config.RelayerConfig.GasOracleSenderPrivateKeys[0], l1ChainID)
	assert.NoError(t, err)
	l1Auth.Value = ether

	l1EthGateway, err := l1gateway.NewL1ETHGateway(base.L1Contracts.L1ETHGateway, l1Cli)
	assert.NoError(t, err)

	value := big.NewInt(100)
	to := common.HexToAddress("0x7363726f6c6c6c02000000000000000000000007")
	tx, err := l1EthGateway.DepositETH0(l1Auth, to, value, big.NewInt(10000))
	assert.NoError(t, err)
	receipt, err := bind.WaitMined(context.Background(), l1Cli, tx)
	assert.NoError(t, err)
	assert.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	db, err := database.NewOrmFactory(base.DBConfig)
	assert.NoError(t, err)
	l1MsgOrm := db.(orm.L1MessageOrm)

	// l1 message wait result.
	ok := utils.TryTimes(60, func() bool {
		msgs, err := l1MsgOrm.GetL1MessagesByStatus(ctypes.MsgConfirmed, 1)
		return err == nil && len(msgs) == 1
	})
	assert.True(t, ok)

	// Check to address balance in l2 chain.
	ok = utils.TryTimes(10, func() bool {
		bls, err := l2Cli.BalanceAt(context.Background(), to, nil)
		return err == nil && bls.Cmp(value) >= 0
	})
	assert.True(t, ok)
}

func testETHWithdraw(t *testing.T) {
	l1Cli, err := base.L1Client()
	assert.NoError(t, err)
	l2Cli, err := base.L2Client()
	assert.NoError(t, err)

	l2EthGateway, err := l2gateway.NewL2ETHGateway(base.L2Contracts.L2ETHGateway, l2Cli)
	assert.NoError(t, err)

	l2ChainID, _ := l2Cli.ChainID(context.Background())
	l2Auth, err := bind.NewKeyedTransactorWithChainID(bridgeApp.Config.L1Config.RelayerConfig.GasOracleSenderPrivateKeys[0], l2ChainID)
	assert.NoError(t, err)
	l2Auth.Value = ether

	bls, err := l2Cli.BalanceAt(context.Background(), common.HexToAddress("0x7363726f6c6c6c02000000000000000000000007"), nil)
	assert.NoError(t, err)
	t.Log("balance in l2chain: ", bls.String())

	to := common.HexToAddress("0x7363726f6c6c6c02000000000000000000000007")
	value := big.NewInt(20)
	tx, err := l2EthGateway.WithdrawETH(l2Auth, to, value, big.NewInt(1000000))
	assert.NoError(t, err)
	receipt, err := bind.WaitMined(context.Background(), l2Cli, tx)
	assert.NoError(t, err)
	assert.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	db, err := database.NewOrmFactory(base.DBConfig)
	assert.NoError(t, err)
	l2MsgOrm := db.(orm.L2MessageOrm)

	// l1 message wait result.
	ok := utils.TryTimes(80, func() bool {
		msgs, err := l2MsgOrm.GetL2Messages(map[string]interface{}{"status": ctypes.MsgConfirmed}, "ORDER BY nonce", "LIMIT 1")
		return err == nil && len(msgs) == 1
	})
	assert.True(t, ok)

	// Check to address balance in l2 chain.
	bls, err = l1Cli.BalanceAt(context.Background(), to, nil)
	assert.NoError(t, err)
	assert.True(t, bls.Cmp(value) >= 0)
}

func translateBatch(batchData *ctypes.BatchData) scrollchain.IScrollChainBatch {
	batch := batchData.Batch
	iBatchData := scrollchain.IScrollChainBatch{
		Blocks:           make([]scrollchain.IScrollChainBlockContext, len(batch.Blocks)),
		PrevStateRoot:    batch.PrevStateRoot,
		NewStateRoot:     batch.NewStateRoot,
		WithdrawTrieRoot: batch.WithdrawTrieRoot,
		BatchIndex:       batch.BatchIndex,
		ParentBatchHash:  batch.ParentBatchHash,
		L2Transactions:   batch.L2Transactions,
	}
	for i, block0 := range batch.Blocks {
		iBatchData.Blocks[i] = scrollchain.IScrollChainBlockContext{
			BlockHash:       block0.BlockHash,
			ParentHash:      block0.ParentHash,
			BlockNumber:     block0.BlockNumber,
			Timestamp:       block0.Timestamp,
			BaseFee:         block0.BaseFee,
			GasLimit:        block0.GasLimit,
			NumTransactions: block0.NumTransactions,
			NumL1Messages:   block0.NumL1Messages,
		}
	}
	return iBatchData
}
