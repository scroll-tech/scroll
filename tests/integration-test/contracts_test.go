package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/rlp"
	"math/big"
	"scroll-tech/common/bytecode/scroll/L1/gateways"
	"testing"

	"github.com/scroll-tech/go-ethereum/core/types"

	"scroll-tech/common/bytecode/erc20"
	"scroll-tech/common/bytecode/greeter"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var (
	// Balance equal to 1e28
	amount = new(big.Int).SetBytes(common.FromHex("204fce5e3e25020000000000"))
	ether  = big.NewInt(1e18)
)

/*
	func TestStandardERC20Gateway(t *testing.T) {
		//base.RunL1Geth(t)
		//base.RunL2Geth(t)
		l1Cli, err := base.L1Client()
		assert.Nil(t, err)

		l2Cli, err := base.L2Client()
		assert.NoError(t, err)

		l1StandardERC20, err := l1gateways.NewL1StandardERC20Gateway(base.L1Contracts.L1StandardERC20Gateway, l1Cli)
		assert.NoError(t, err)

		l2StandardERC20, err := l2gateways.NewL2StandardERC20Gateway(base.L2Contracts.L2StandardERC20Gateway, l2Cli)
		assert.NoError(t, err)

		// check l2 standard erc20 address.
		l2ERC20_0, err := l1StandardERC20.GetL2ERC20Address(nil, base.L1Contracts.L1WETH)
		assert.NoError(t, err)
		l2ERC20_1, err := l2StandardERC20.GetL2ERC20Address(nil, base.L1Contracts.L1WETH)
		assert.NoError(t, err)
		assert.Equal(t, l2ERC20_0, l2ERC20_1)
	}
*/
func TestERC20(t *testing.T) {
	//base.RunL2Geth(t)
	l2Cli, err := base.L2Client()
	assert.Nil(t, err)

	token, err := erc20.NewERC20Mock(base.ERC20, l2Cli)
	assert.NoError(t, err)

	chainID, _ := l2Cli.ChainID(context.Background())
	auth, err := bind.NewKeyedTransactorWithChainID(bridgeApp.Config.L2Config.RelayerConfig.GasOracleSenderPrivateKeys[0], chainID)
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
	bind.WaitMined(context.Background(), l2Cli, tx)

	authBls1, err := token.BalanceOf(nil, auth.From)
	assert.NoError(t, err)

	tokenBls1, err := token.BalanceOf(nil, to)
	assert.NoError(t, err)

	// check balance.
	assert.Equal(t, authBls0.Int64(), authBls1.Add(authBls1, value).Int64())
	assert.Equal(t, tokenBls1.Int64(), tokenBls0.Add(tokenBls0, value).Int64())
}

func TestGreeter(t *testing.T) {
	//base.RunL2Geth(t)
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

	res, err := token.Retrieve(nil)
	assert.NoError(t, err)
	assert.Equal(t, val.String(), res.String())
}

/*func TestMintERC20(t *testing.T) {
	t.Log(base.L1Contracts.L1WETH.String())
	//base.RunL1Geth(t)
	l1Cli, err := ethclient.Dial("http://localhost:8545") //base.L1Client()
	assert.Nil(t, err)

	l1StandardERC20, err := l1gateways.NewL1StandardERC20Gateway(base.L1Contracts.L1StandardERC20Gateway, l1Cli)
	assert.NoError(t, err)

	L1ERC20, err := erc20.NewERC20Mock(base.L1Contracts.L1WETH, l1Cli)
	assert.NoError(t, err)

	chainID, _ := l1Cli.ChainID(context.Background())
	l1Auth, err := bind.NewKeyedTransactorWithChainID(bridgeApp.Config.L2Config.RelayerConfig.GasOracleSenderPrivateKeys[0], chainID)
	assert.NoError(t, err)
	l1Auth.GasLimit = 1000000000

	// check init balance in erc20 contract.
	val, err := L1ERC20.BalanceOf(nil, l1Auth.From)
	assert.NoError(t, err)
	assert.Equal(t, amount, val)

	// Approve for l1 standard erc20 gateway.
	allow, err := L1ERC20.Allowance(nil, l1Auth.From, base.L1Contracts.L1StandardERC20Gateway)
	assert.NoError(t, err)
	assert.Equal(t, amount, allow)

	tx, err := l1StandardERC20.DepositERC20(l1Auth, base.L1Contracts.L1WETH, ether, big.NewInt(10000000000))
	assert.NoError(t, err)
	receipt, err := bind.WaitMined(context.Background(), l1Cli, tx)
	assert.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
}*/

//func TestCheckL1Address(t *testing.T) {
//	//base.RunL1Geth(t)
//	//base.RunL2Geth(t)
//	l1Cli, err := ethclient.Dial("ws://localhost:11954") //base.L1Client()
//	assert.Nil(t, err)
//
//	l2Cli, err := ethclient.Dial("ws://localhost:20869") //base.L2Client()
//	assert.NoError(t, err)
//
//	l1StandardERC20, err := l1gateways.NewL1StandardERC20Gateway(base.L1Contracts.L1StandardERC20Gateway, l1Cli)
//	assert.NoError(t, err)
//
//	l2ChainID, _ := l2Cli.ChainID(context.Background())
//	l2StandardERC20, err := l2gateways.NewL2StandardERC20Gateway(base.L2Contracts.L2StandardERC20Gateway, l2Cli)
//	assert.NoError(t, err)
//	l2factory, err := libraries.NewScrollStandardERC20Factory(base.L2Contracts.L2ScrollStandardERC20Factory, l2Cli)
//	assert.NoError(t, err)
//	l2Auth, err := bind.NewKeyedTransactorWithChainID(bridgeApp.Config.L1Config.RelayerConfig.GasOracleSenderPrivateKeys[0], l2ChainID)
//	assert.NoError(t, err)
//
//	tx, err := l2factory.DeployL2Token(l2Auth, base.L2Contracts.L2StandardERC20Gateway, base.L1Contracts.L1WETH)
//	assert.NoError(t, err)
//	_, err = bind.WaitMined(context.Background(), l2Cli, tx)
//	assert.NoError(t, err)
//
//	addr, err := l2factory.ComputeL2TokenAddress(nil, base.L2Contracts.L2StandardERC20Gateway, base.L1Contracts.L1WETH)
//	assert.NoError(t, err)
//	t.Log("l2 er20 address: ", addr.String())
//
//	code, err := l2Cli.CodeAt(context.Background(), addr, nil)
//	assert.NoError(t, err)
//	t.Log("code: ", common.Bytes2Hex(code))
//
//	/*l2ERC20, err := libraries.NewScrollStandardERC20(addr, l2Cli)
//	assert.NoError(t, err)
//	bls, err := l2ERC20.BalanceOf(nil, l2Auth.From)
//	assert.NoError(t, err)
//	t.Log("l2 chain balance: ", bls.Int64())*/
//
//	l2ERC20_l1, err := l1StandardERC20.GetL2ERC20Address(nil, base.L1Contracts.L1WETH)
//	assert.NoError(t, err)
//	t.Log(l2ERC20_l1.String())
//
//	l2ERC20_l2, err := l2StandardERC20.GetL2ERC20Address(nil, base.L1Contracts.L1WETH)
//	assert.NoError(t, err)
//	t.Log(l2ERC20_l2.String())
//
//	// check l2 erc20 address.
//	assert.Equal(t, l2ERC20_l1, l2ERC20_l2)
//}

//func TestStandardERC20Deposit(t *testing.T) {
//	//base.RunImages(t)
//	// Reset db.
//	l1Cli, err := base.L1Client()
//	assert.Nil(t, err)
//	l2Cli, err := base.L2Client()
//	assert.NoError(t, err)
//	//assert.NoError(t, migrate.ResetDB(base.DBClient(t)))
//	t.Log("bridge config file: ", bridgeApp.BridgeConfigFile)
//	//t.Log("coordinator config file: ", coordinatorApp.CoordinatorConfigFile)
//	//t.Log("roller config file: ", rollerApp.RollerConfigFile)
//
//	// check result.
//	//db, err := database.NewOrmFactory(base.DBConfig)
//	//assert.NoError(t, err)
//
//	l1StandardERC20, err := l1gateways.NewL1StandardERC20Gateway(base.L1Contracts.L1StandardERC20Gateway, l1Cli)
//	assert.NoError(t, err)
//	l1ChainID, _ := l1Cli.ChainID(context.Background())
//	l1Auth, err := bind.NewKeyedTransactorWithChainID(bridgeApp.Config.L2Config.RelayerConfig.GasOracleSenderPrivateKeys[0], l1ChainID)
//	assert.NoError(t, err)
//	l1Auth.GasLimit = 100000000
//
//	// Run event_watcher process.
//	//bridgeApp.RunApp(t, utils.EventWatcherApp)
//
//	// l1 => l2 deposit transfer.
//	tx, err := l1StandardERC20.DepositERC20(l1Auth, base.L1Contracts.L1WETH, ether, big.NewInt(100000000))
//	assert.NoError(t, err)
//	receipt, err := bind.WaitMined(context.Background(), l1Cli, tx)
//	assert.NoError(t, err)
//	assert.True(t, receipt.Status == types.ReceiptStatusSuccessful)
//	t.Log("block number: ", receipt.BlockNumber.Int64())
//
//	/*
//		var (
//			l1MsgOrm = db.(orm.L1MessageOrm)
//			l1Msgs   []*stypes.L1Message
//		)
//		// Catch event result.
//		utils.TryTimes(30, func() bool {
//			l1Msgs, err = l1MsgOrm.GetL1MessagesByStatus(stypes.MsgPending, 1)
//			return err == nil && len(l1Msgs) != 0
//		})
//		assert.Equal(t, 1, len(l1Msgs))
//		assert.Equal(t, receipt.BlockNumber.Uint64(), l1Msgs[0].Height)
//		assert.Equal(t, receipt.TxHash.String(), l1Msgs[0].Layer1Hash)
//	*/
//
//	/*l2factory, err := libraries.NewScrollStandardERC20Factory(base.L2Contracts.L2ScrollStandardERC20Factory, l2Cli)
//	assert.NoError(t, err)
//
//	owner, err := l2factory.Owner(nil)
//	assert.NoError(t, err)
//	t.Log("l2 factory owner: ", owner.String())*/
//
//	for i := 0; i < 1000; i++ {
//		bls, err := l2Cli.BalanceAt(context.Background(), l1Auth.From, nil)
//		if err == nil {
//			t.Log(bls.Int64())
//		}
//	}
//
//	bridgeApp.WaitExit()
//}

func TestETHDeposit(t *testing.T) {
	l1Cli, err := base.L1Client()
	assert.NoError(t, err)
	t.Log("bridge config file: ", bridgeApp.BridgeConfigFile)
	t.Log("coordinator config file: ", coordinatorApp.CoordinatorConfigFile)
	t.Log("roller config file: ", rollerApp.RollerConfigFile)

	l1ChainID, _ := l1Cli.ChainID(context.Background())
	l1Auth, err := bind.NewKeyedTransactorWithChainID(bridgeApp.Config.L2Config.RelayerConfig.GasOracleSenderPrivateKeys[0], l1ChainID)
	assert.NoError(t, err)
	l1Auth.GasLimit = 10000000

	l1EthGateway, err := gateways.NewL1ETHGateway(base.L1Contracts.L1ETHGateway, l1Cli)
	assert.NoError(t, err)
	l1Auth.Value = ether
	to := common.HexToAddress("0x7363726f6c6c6c02000000000000000000000007")
	tx, err := l1EthGateway.DepositETH0(l1Auth, to, big.NewInt(1), big.NewInt(10000))
	assert.NoError(t, err)
	receipt, err := bind.WaitMined(context.Background(), l1Cli, tx)
	assert.NoError(t, err)
	assert.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
}

func TestCC(t *testing.T) {
	l2Cli, err := base.L2Client()
	assert.NoError(t, err)

	var (
		number      int64 = 1
		parentBlock *types.Block
	)
	latestHeader, err := l2Cli.HeaderByNumber(context.Background(), nil)
	assert.NoError(t, err)
	for ; number < latestHeader.Number.Int64(); number++ {
		block, err := l2Cli.BlockByNumber(context.Background(), big.NewInt(number))
		assert.NoError(t, err)
		if parentBlock == nil {
			parentBlock = block
		} else {
			if parentBlock.Hash() != block.ParentHash() {
				header := block.Header()
				data, _ := json.Marshal(header)
				fmt.Println("header json encode ----------- ", "number: ", header.Number.Uint64(), "content: ", string(data))
				data, _ = rlp.EncodeToBytes(header)
				fmt.Println("header rlp encode ----------- ", "number: ", header.Number.Uint64(), "content: ", hexutil.Encode(data))

			}
			parentBlock = block
		}
	}
}

func TestXX(t *testing.T) {
	var tmp = big.NewInt(0)
	data, err := rlp.EncodeToBytes(tmp)
	assert.NoError(t, err)
	t.Log(hexutil.Encode(data))

	var tmp1 *big.Int
	data, err = rlp.EncodeToBytes(tmp1)
	assert.NoError(t, err)
	t.Log(hexutil.Encode(data))
}
