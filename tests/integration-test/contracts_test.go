package integration_test

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/bytecode/erc20"
	"scroll-tech/common/bytecode/greeter"
)

var (
	erc20Address   = common.HexToAddress("0x7363726f6c6c6c20000000000000000000000014")
	greeterAddress = common.HexToAddress("0x7363726f6c6c6c20000000000000000000000015")
)

func testERC20(t *testing.T) {
	assert.NoError(t, testApps.StartL2GethContainer())
	time.Sleep(time.Second * 3)

	l2Cli, err := testApps.GetL2GethClient()
	assert.Nil(t, err)

	token, err := erc20.NewERC20Mock(erc20Address, l2Cli)
	assert.NoError(t, err)
	privKey, err := crypto.ToECDSA(common.FromHex("1212121212121212121212121212121212121212121212121212121212121212"))
	assert.NoError(t, err)

	chainID, err := l2Cli.ChainID(context.Background())
	assert.NoError(t, err)
	auth, err := bind.NewKeyedTransactorWithChainID(privKey, chainID)
	assert.NoError(t, err)

	authBls0, err := token.BalanceOf(nil, auth.From)
	assert.NoError(t, err)

	tokenBls0, err := token.BalanceOf(nil, erc20Address)
	assert.NoError(t, err)

	// create tx to transfer balance.
	value := big.NewInt(1000)
	tx, err := token.Transfer(auth, erc20Address, value)
	assert.NoError(t, err)
	_, err = bind.WaitMined(context.Background(), l2Cli, tx)
	assert.NoError(t, err)

	authBls1, err := token.BalanceOf(nil, auth.From)
	assert.NoError(t, err)

	tokenBls1, err := token.BalanceOf(nil, erc20Address)
	assert.NoError(t, err)

	// check balance.
	assert.Equal(t, authBls0.Int64(), authBls1.Add(authBls1, value).Int64())
	assert.Equal(t, tokenBls1.Int64(), tokenBls0.Add(tokenBls0, value).Int64())
}

func testGreeter(t *testing.T) {
	assert.NoError(t, testApps.StartL2GethContainer())
	l2Cli, err := testApps.GetL2GethClient()
	assert.Nil(t, err)

	chainID, err := l2Cli.ChainID(context.Background())
	assert.NoError(t, err)
	pKey, err := crypto.ToECDSA(common.FromHex(rollupApp.Config.L2Config.RelayerConfig.CommitSenderSignerConfig.PrivateKeySignerConfig.PrivateKey))
	assert.NoError(t, err)
	auth, err := bind.NewKeyedTransactorWithChainID(pKey, chainID)
	assert.NoError(t, err)

	token, err := greeter.NewGreeter(greeterAddress, l2Cli)
	assert.NoError(t, err)

	val := big.NewInt(100)
	tx, err := token.SetValue(auth, val)
	assert.NoError(t, err)
	_, err = bind.WaitMined(context.Background(), l2Cli, tx)

	res, err := token.Retrieve(nil)
	assert.NoError(t, err)
	assert.Equal(t, val.String(), res.String())
}
