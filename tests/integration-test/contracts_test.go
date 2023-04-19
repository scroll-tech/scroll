package integration

import (
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/bytecode/erc20"
)

var (
	daoAddress     = common.HexToAddress("0x7363726f6c6c6c20000000000000000000000013")
	erc20Address   = common.HexToAddress("7363726f6c6c6c20000000000000000000000014")
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
	// Start l2geth docker.
	base.RunL2Geth(t)

	l2Cli, err := base.L2Client()
	assert.Nil(t, err)
	token, err := erc20.NewERC20Mock(erc20Address, l2Cli)
	assert.NoError(t, err)
	bls, err := token.BalanceOf(nil, erc20Address)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), bls.Int64())
}
