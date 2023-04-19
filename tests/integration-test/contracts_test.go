package integration

import (
	"context"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"math/big"
	"scroll-tech/common/bytecode/dao"
	"scroll-tech/common/bytecode/greeter"
	"scroll-tech/common/bytecode/nft"
	"scroll-tech/common/bytecode/sushi"
	"scroll-tech/common/bytecode/vote"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/bytecode/erc20"
)

var (
	daoAddress     = common.HexToAddress("0x7363726f6c6c6c20000000000000000000000013")
	erc20Address   = common.HexToAddress("0x7363726f6c6c6c20000000000000000000000014")
	greeterAddress = common.HexToAddress("0x7363726f6c6c6c20000000000000000000000015")
	nftAddress     = common.HexToAddress("0x7363726f6c6c6c20000000000000000000000016")

	sushiTokenAddress      = common.HexToAddress("0x7363726f6c6c6c20000000000000000000000017")
	sushiMasterchefAddress = common.HexToAddress("0x7363726f6c6c6c20000000000000000000000018")

	voteAddress               = common.HexToAddress("0x7363726f6c6c6c20000000000000000000000019")
	uniswapV2FactoryAddress   = common.HexToAddress("0x7363726f6c6c6c20000000000000000000000020")
	uniswapV2Router02Address  = common.HexToAddress("0x7363726f6c6c6c20000000000000000000000021")
	uniswapV2MulticallAddress = common.HexToAddress("0x7363726f6c6c6c20000000000000000000000022")
	uniswapV2WETH9            = common.HexToAddress("0x7363726f6c6c6c20000000000000000000000023")
)

func TestERC20(t *testing.T) {
	base.RunL2Geth(t)
	l2Cli, err := base.L2Client()
	assert.Nil(t, err)
	token, err := erc20.NewERC20Mock(erc20Address, l2Cli)
	assert.NoError(t, err)

	auth, err := bind.NewKeyedTransactorWithChainID(bridgeApp.Config.L2Config.RelayerConfig.MessageSenderPrivateKeys[0], base.L2gethImg.ChainID())
	assert.NoError(t, err)

	authBls0, err := token.BalanceOf(nil, auth.From)
	assert.NoError(t, err)

	tokenBls0, err := token.BalanceOf(nil, erc20Address)
	assert.NoError(t, err)

	// create tx to transfer balance.
	bls := big.NewInt(1000)
	tx, err := token.Transfer(auth, erc20Address, bls)
	assert.NoError(t, err)
	bind.WaitMined(context.Background(), l2Cli, tx)

	authBls1, err := token.BalanceOf(nil, auth.From)
	assert.NoError(t, err)

	tokenBls1, err := token.BalanceOf(nil, erc20Address)
	assert.NoError(t, err)

	// check balance.
	assert.Equal(t, authBls0.Int64(), authBls1.Add(authBls1, bls).Int64())
	assert.Equal(t, tokenBls1.Int64(), tokenBls0.Add(tokenBls0, bls).Int64())
}

func TestVote(t *testing.T) {
	base.RunL2Geth(t)
	l2Cli, err := base.L2Client()
	assert.Nil(t, err)

	auth, err := bind.NewKeyedTransactorWithChainID(bridgeApp.Config.L2Config.RelayerConfig.MessageSenderPrivateKeys[0], base.L2gethImg.ChainID())
	assert.NoError(t, err)

	mockVote, err := vote.NewVotesMock(voteAddress, l2Cli)
	assert.NoError(t, err)

	target := common.HexToAddress("0xb7C0c58702D0781C0e2eB3aaE301E4c340073448")
	voteId := big.NewInt(1000)
	tx, err := mockVote.Mint(auth, target, voteId)
	assert.NoError(t, err)
	_, err = bind.WaitMined(context.Background(), l2Cli, tx)
	assert.NoError(t, err)

	count, err := mockVote.GetTotalSupply(nil)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count.Int64())

	tx, err = mockVote.Delegate(auth, target)
	assert.NoError(t, err)
	_, err = bind.WaitMined(context.Background(), l2Cli, tx)
	assert.NoError(t, err)

	addr, err := mockVote.Delegates(nil, auth.From)
	assert.NoError(t, err)
	assert.Equal(t, target.String(), addr.String())
}

func TestDao(t *testing.T) {
	base.RunL2Geth(t)
	l2Cli, err := base.L2Client()
	assert.Nil(t, err)

	auth, err := bind.NewKeyedTransactorWithChainID(bridgeApp.Config.L2Config.RelayerConfig.MessageSenderPrivateKeys[0], base.L2gethImg.ChainID())
	assert.NoError(t, err)
	auth.GasPrice = big.NewInt(1108583800)
	auth.GasLimit = 11529940

	token, err := dao.NewGovernorMock(daoAddress, l2Cli)
	assert.NoError(t, err)

	target := common.HexToAddress("0xb7C0c58702D0781C0e2eB3aaE301E4c340073448")
	value := big.NewInt(1)
	tx, err := token.Propose(auth, []common.Address{target}, []*big.Int{value}, nil, "dao propose test")
	assert.NoError(t, err)
	_, err = bind.WaitMined(context.Background(), l2Cli, tx)
	assert.NoError(t, err)
	tx, err = token.Cancel(auth, []common.Address{target}, []*big.Int{value}, nil, common.Hash{})
	assert.NoError(t, err)
	_, err = bind.WaitMined(context.Background(), l2Cli, tx)
}

func TestGreeter(t *testing.T) {
	base.RunL2Geth(t)
	l2Cli, err := base.L2Client()
	assert.Nil(t, err)

	auth, err := bind.NewKeyedTransactorWithChainID(bridgeApp.Config.L2Config.RelayerConfig.MessageSenderPrivateKeys[0], base.L2gethImg.ChainID())
	assert.NoError(t, err)

	token, err := greeter.NewGreeter(greeterAddress, l2Cli)
	assert.NoError(t, err)

	val := big.NewInt(100)
	tx, err := token.SetValue(auth, val)
	assert.NoError(t, err)
	_, err = bind.WaitMined(context.Background(), l2Cli, tx)

	bls, err := token.Retrieve(nil)
	assert.NoError(t, err)
	assert.Equal(t, val.String(), bls.String())
}

func TestNFT(t *testing.T) {
	base.RunL2Geth(t)
	l2Cli, err := base.L2Client()
	assert.Nil(t, err)

	auth, err := bind.NewKeyedTransactorWithChainID(bridgeApp.Config.L2Config.RelayerConfig.MessageSenderPrivateKeys[0], base.L2gethImg.ChainID())
	assert.NoError(t, err)

	token, err := nft.NewERC721Mock(nftAddress, l2Cli)
	assert.NoError(t, err)

	tokenID := big.NewInt(100)
	tx, err := token.Mint(auth, auth.From, tokenID)
	assert.NoError(t, err)
	_, err = bind.WaitMined(context.Background(), l2Cli, tx)
	assert.NoError(t, err)

	exist, err := token.Exists(nil, tokenID)
	assert.True(t, err == nil && exist)

	target := common.HexToAddress("0xb7C0c58702D0781C0e2eB3aaE301E4c340073448")
	tx, err = token.Approve(auth, target, tokenID)
	assert.NoError(t, err)
	_, err = bind.WaitMined(context.Background(), l2Cli, tx)
	assert.NoError(t, err)

	tx, err = token.TransferFrom(auth, auth.From, target, tokenID)
	assert.NoError(t, err)
	_, err = bind.WaitMined(context.Background(), l2Cli, tx)
	assert.NoError(t, err)

	addr, err := token.OwnerOf(nil, tokenID)
	assert.NoError(t, err)
	assert.Equal(t, target.String(), addr.String())
}

func TestSushi(t *testing.T) {
	l2Cli, err := ethclient.Dial("ws://localhost:20869") //base.L2Client()
	assert.NoError(t, err)

	auth, err := bind.NewKeyedTransactorWithChainID(bridgeApp.Config.L2Config.RelayerConfig.MessageSenderPrivateKeys[0], base.L2gethImg.ChainID())
	assert.NoError(t, err)
	auth.GasPrice = big.NewInt(1108583800)
	auth.GasLimit = 11529940

	// sushi token handler
	sushiToken, err := sushi.NewSushiToken(sushiTokenAddress, l2Cli)
	assert.NoError(t, err)

	// master chef handler
	chef, err := sushi.NewMasterChef(sushiTokenAddress, l2Cli)
	assert.NoError(t, err)

	ether := big.NewInt(1e18)

	amount := ether
	tx, err := sushiToken.Mint(auth, auth.From, amount)
	assert.NoError(t, err)

	allocPoint := ether
	tx, err = chef.Add(auth, allocPoint, sushiTokenAddress, true)
	assert.NoError(t, err)
	_, err = bind.WaitMined(context.Background(), l2Cli, tx)
	assert.NoError(t, err)

	t.Log("!!!!!!!!!!!!!!!!!!!!must wait here to have pool!!!!!!!!!!!!!!!!!!!!")

	pid, err := chef.PoolLength(&bind.CallOpts{Pending: true})
	assert.NoError(t, err)
	pid.Sub(pid, big.NewInt(1))

	tx, err = chef.Set(auth, pid, allocPoint, true)
	assert.NoError(t, err)
	_, err = bind.WaitMined(context.Background(), l2Cli, tx)
	assert.NoError(t, err)

	// check pointInfo
	poolInfo, err := chef.PoolInfo(nil, pid)
	assert.NoError(t, err)
	assert.Equal(t, 0, poolInfo.AllocPoint.Cmp(allocPoint))

	// approve chef deposit
	tx, err = sushiToken.Approve(auth, sushiTokenAddress, amount)
	assert.NoError(t, err)
	_, err = bind.WaitMined(context.Background(), l2Cli, tx)
	assert.NoError(t, err)

	t.Log("!!!!!!!!!!!!!!!!!!!!must wait here for allowance!!!!!!!!!!!!!!!!!!!!")

	// deposit amount to chef
	tx, err = chef.Deposit(auth, pid, amount)
	assert.NoError(t, err)
	_, err = bind.WaitMined(context.Background(), l2Cli, tx)
	assert.NoError(t, err)

	// userInfo's amount is equal to auth0's
	userInfo, err := chef.UserInfo(nil, pid, auth.From)
	assert.NoError(t, err)
	assert.Equal(t, 0, userInfo.Amount.Cmp(amount))

	bls, err := sushiToken.BalanceOf(nil, auth.From)
	assert.NoError(t, err)
	assert.Equal(t, 0, bls.Cmp(big.NewInt(0)))

	// change sushiToken's owner to masterChef.
	tx, err = sushiToken.TransferOwnership(auth, sushiTokenAddress)
	assert.NoError(t, err)
	_, err = bind.WaitMined(context.Background(), l2Cli, tx)
	assert.NoError(t, err)

	t.Log("!!!!!!!!!!!!!!!!!!!!must wait here for ownership!!!!!!!!!!!!!!!!!!!!")

	// withdraw amount from chef
	tx, err = chef.Withdraw(auth, pid, amount)
	assert.NoError(t, err)
	_, err = bind.WaitMined(context.Background(), l2Cli, tx)
	assert.NoError(t, err)

	t.Log("-------------------testing withdraw--------------------------------------")

	bls, err = sushiToken.BalanceOf(nil, auth.From)
	assert.NoError(t, err)
	assert.Equal(t, 0, bls.Cmp(amount))
}
