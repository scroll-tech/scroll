package integration_test

import (
	"context"
	"math/big"
	"scroll-tech/common/utils"
	"scroll-tech/database"
	"scroll-tech/database/migrate"
	"scroll-tech/database/orm"
	"testing"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/bytecode/erc20"
	"scroll-tech/common/bytecode/greeter"
	l1gateway "scroll-tech/common/bytecode/scroll/L1/gateways"
	ctypes "scroll-tech/common/types"
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

func TestETHDeposit(t *testing.T) {
	base.RunImages(t)
	// Reset db.
	assert.NoError(t, migrate.ResetDB(base.DBClient(t)))

	l1Cli, err := base.L1Client()
	assert.NoError(t, err)
	l2Cli, err := base.L2Client()
	assert.NoError(t, err)

	// Start event watcher.
	bridgeApp.RunApp(t, utils.EventWatcherApp)
	// Start gas price oracle.
	bridgeApp.RunApp(t, utils.GasOracleApp)
	// Start message relayer.
	bridgeApp.RunApp(t, utils.MessageRelayerApp)

	l1ChainID, _ := l1Cli.ChainID(context.Background())
	l1Auth, err := bind.NewKeyedTransactorWithChainID(bridgeApp.Config.L2Config.RelayerConfig.GasOracleSenderPrivateKeys[0], l1ChainID)
	assert.NoError(t, err)

	l1EthGateway, err := l1gateway.NewL1ETHGateway(base.L1Contracts.L1ETHGateway, l1Cli)
	assert.NoError(t, err)

	l1Auth.Value = ether
	to := common.HexToAddress("0x7363726f6c6c6c02000000000000000000000007")
	tx, err := l1EthGateway.DepositETH0(l1Auth, to, big.NewInt(1), big.NewInt(10000))
	assert.NoError(t, err)
	receipt, err := bind.WaitMined(context.Background(), l1Cli, tx)
	assert.NoError(t, err)
	assert.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	db, err := database.NewOrmFactory(base.DBConfig)
	assert.NoError(t, err)
	l1MsgOrm := db.(orm.L1MessageOrm)

	var msgs []*ctypes.L1Message
	// l1 message wait result.
	utils.TryTimes(60, func() bool {
		msgs, err = l1MsgOrm.GetL1MessagesByStatus(ctypes.MsgConfirmed, 1)
		return err == nil && len(msgs) == 1
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(msgs))
	assert.Equal(t, tx.Hash().String(), msgs[0].Layer1Hash)

	// Check to address balance in l2 chain.
	bls, err := l2Cli.BalanceAt(context.Background(), to, nil)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), bls.Int64())
}
