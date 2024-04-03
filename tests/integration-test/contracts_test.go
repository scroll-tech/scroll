package integration_test

import (
	"context"
	"math/big"
	"testing"

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

func TestERC20(t *testing.T) {
	l2Cli, err := base.L2Client()
	assert.Nil(t, err)

	token, err := erc20.NewERC20Mock(erc20Address, l2Cli)
	assert.NoError(t, err)
	privKey, err := crypto.ToECDSA(common.FromHex("1212121212121212121212121212121212121212121212121212121212121212"))
	assert.NoError(t, err)

	auth, err := bind.NewKeyedTransactorWithChainID(privKey, base.L2gethImg.ChainID())
	assert.NoError(t, err)

	authBls0, err := token.BalanceOf(nil, auth.From)
	assert.NoError(t, err)

	tokenBls0, err := token.BalanceOf(nil, erc20Address)
	assert.NoError(t, err)

	// create tx to transfer balance.
	value := big.NewInt(1000)
	tx, err := token.Transfer(auth, erc20Address, value)
	assert.NoError(t, err)
	bind.WaitMined(context.Background(), l2Cli, tx)

	authBls1, err := token.BalanceOf(nil, auth.From)
	assert.NoError(t, err)

	tokenBls1, err := token.BalanceOf(nil, erc20Address)
	assert.NoError(t, err)

	// check balance.
	assert.Equal(t, authBls0.Int64(), authBls1.Add(authBls1, value).Int64())
	assert.Equal(t, tokenBls1.Int64(), tokenBls0.Add(tokenBls0, value).Int64())
}

func TestGreeter(t *testing.T) {
	l2Cli, err := base.L2Client()
	assert.Nil(t, err)

	auth, err := bind.NewKeyedTransactorWithChainID(rollupApp.Config.L2Config.RelayerConfig.CommitSenderPrivateKey, base.L2gethImg.ChainID())
	assert.NoError(t, err)

	token, err := greeter.NewGreeter(greeterAddress, l2Cli)
	assert.NoError(t, err)

	val := big.NewInt(100)
	tx, err := token.SetValue(auth, val)
	assert.NoError(t, err)

	done := make(chan struct{})
	go func() {
		_, err = bind.WaitMined(context.Background(), l2Cli, tx)
		close(done)
	}()

	select {
	case <-done:
		res, err := token.Retrieve(nil)
		assert.NoError(t, err)
		assert.Equal(t, val.String(), res.String())
	case <-time.After(time.Second * 10): // Timeout after 10 seconds
		t.Fatal("timeout waiting for transaction to be mined")
	}
}
