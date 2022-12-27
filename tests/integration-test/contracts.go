package integration

import (
	"context"
	"crypto/ecdsa"
	"math"
	"math/big"
	"math/rand"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"

	bridgeConfig "scroll-tech/bridge/config"
	"scroll-tech/bridge/sender"
	"scroll-tech/common/utils"
	"scroll-tech/contracts/dao"
	"scroll-tech/contracts/erc20"
	"scroll-tech/contracts/greeter"
	"scroll-tech/contracts/nft"
	"scroll-tech/contracts/sushi"
	"scroll-tech/contracts/uniswap/factory"
	"scroll-tech/contracts/uniswap/router"
	"scroll-tech/contracts/uniswap/weth9"
	"scroll-tech/contracts/vote"
)

func native(ctx context.Context, to common.Address, value *big.Int) error {
	// create and send native tx.
	newSender, err := sender.NewSender(ctx, &bridgeConfig.SenderConfig{
		Endpoint:            l2gethImg.Endpoint(),
		CheckPendingTime:    3,
		EscalateBlocks:      100,
		Confirmations:       0,
		EscalateMultipleNum: 11,
		EscalateMultipleDen: 10,
		TxType:              "DynamicFeeTx",
	}, []*ecdsa.PrivateKey{privkey})
	//to = common.HexToAddress("0x1c5a77d9fa7ef466951b2f01f724bca3a5820b63")
	_, err = newSender.SendTransaction("native_01", &to, value, nil)
	<-newSender.ConfirmChan()
	return err
}

func newERC20(ctx context.Context, client *ethclient.Client, root, auth *bind.TransactOpts) error {
	_, tx, erc20Token, err := erc20.DeployERC20Template(root, client, root.From, root.From, "WETH coin", "WETH", 18)
	if err != nil {
		return err
	}

	tx, err = erc20Token.Mint(root, root.From, big.NewInt(1e4))
	if err != nil {
		return err
	}

	// erc20 transfer
	tx, err = erc20Token.Transfer(root, auth.From, big.NewInt(1000))
	if err != nil {
		return err
	}

	for i := 0; i < 10; i++ {
		// erc20 transfer
		tx, err = erc20Token.Transfer(root, auth.From, big.NewInt(1000))
		if err != nil {
			return err
		}
	}
	_, err = bind.WaitMined(ctx, client, tx)
	return err
}

func newGreeter(ctx context.Context, client *ethclient.Client, root *bind.TransactOpts) error {
	_, tx, token, err := greeter.DeployGreeter(root, client, big.NewInt(10))
	if err != nil {
		return err
	}

	tx, err = token.SetValue(root, big.NewInt(10))
	if err != nil {
		return err
	}
	_, err = bind.WaitMined(ctx, client, tx)
	return err
}

func newNft(ctx context.Context, client *ethclient.Client, root, auth *bind.TransactOpts) error {
	_, tx, token, err := nft.DeployERC721Mock(root, client, "ERC721 coin", "ERC721")
	if err != nil {
		return err
	}

	tokenId := big.NewInt(rand.Int63())
	tx, err = token.Mint(root, root.From, tokenId)
	if err != nil {
		return err
	}

	tx, err = token.TransferFrom(root, root.From, auth.From, tokenId)
	if err != nil {
		return err
	}

	tx, err = token.Burn(auth, tokenId)
	if err != nil {
		return err
	}
	_, err = bind.WaitMined(ctx, client, tx)
	return err
}

func newSushi(ctx context.Context, client *ethclient.Client, root *bind.TransactOpts) error {
	sushiAddr, tx, sushiToken, err := sushi.DeploySushiToken(root, client)
	if err != nil {
		return err
	}

	chefAddr, tx, chefToken, err := sushi.DeployMasterChef(root, client, sushiAddr, root.From, big.NewInt(1), big.NewInt(1), big.NewInt(math.MaxInt))
	if err != nil {
		return err
	}

	amount := big.NewInt(1e18)
	tx, err = sushiToken.Mint(root, root.From, amount)
	if err != nil {
		return err
	}

	allocPoint := utils.Ether
	tx, err = chefToken.Add(root, allocPoint, sushiAddr, true)
	if err != nil {
		return err
	}

	pid, err := chefToken.PoolLength(&bind.CallOpts{Pending: true})
	if err != nil {
		return err
	}
	pid.Sub(pid, big.NewInt(1))
	tx, err = chefToken.Set(root, pid, allocPoint, true)
	if err != nil {
		return err
	}

	tx, err = sushiToken.Approve(root, chefAddr, amount)
	if err != nil {
		return err
	}

	// deposit amount to chef
	tx, err = chefToken.Deposit(root, pid, amount)
	if err != nil {
		return err
	}

	// change sushiToken's owner to masterChef.
	tx, err = sushiToken.TransferOwnership(root, chefAddr)
	if err != nil {
		return err
	}

	// withdraw amount from chef
	tx, err = chefToken.Withdraw(root, pid, amount)
	if err != nil {
		return err
	}
	_, err = bind.WaitMined(ctx, client, tx)
	return err
}

func newDao(ctx context.Context, client *ethclient.Client, root *bind.TransactOpts) error {
	voteAddr, tx, _, err := vote.DeployVotesMock(root, client, "vote v2")
	if err != nil {
		return err
	}

	_, tx, daoToken, err := dao.DeployGovernorMock(root, client, "governor mock", voteAddr, big.NewInt(1), big.NewInt(1), big.NewInt(100))
	if err != nil {
		return err
	}

	callData := [][]byte{big.NewInt(1).Bytes()}
	target := common.BigToAddress(big.NewInt(1))
	value := big.NewInt(1)
	description := "dao propose test"
	tx, err = daoToken.Propose(root, []common.Address{target}, []*big.Int{value}, callData, description)
	if err != nil {
		return err
	}

	salt := crypto.Keccak256Hash([]byte(description))
	tx, err = daoToken.Cancel(root, []common.Address{target}, []*big.Int{value}, callData, salt)
	if err != nil {
		return err
	}
	_, err = bind.WaitMined(ctx, client, tx)
	return err
}

func newUniswapv2(ctx context.Context, client *ethclient.Client, root, auth *bind.TransactOpts) error {
	wethAddr, tx, wToken, err := weth9.DeployWETH9(root, client)
	if err != nil {
		return err
	}

	// deploy factory
	fAddr, tx, fToken, err := factory.DeployUniswapV2Factory(root, client, root.From)
	if err != nil {
		return err
	}

	// deploy router
	rAddr, tx, rToken, err := router.DeployUniswapV2Router02(root, client, fAddr, wethAddr)
	if err != nil {
		return err
	}

	btcAddr, tx, btcToken, err := erc20.DeployERC20Template(root, client, auth.From, auth.From, "BTC coin", "BTC", 18)
	if err != nil {
		return err
	}

	// init balance
	auth.GasPrice = big.NewInt(1108583800)
	auth.GasLimit = 11529000
	originVal := big.NewInt(1).Mul(big.NewInt(3e3), utils.Ether)
	tx, err = wToken.Deposit(auth)
	tx, err = btcToken.Mint(auth, auth.From, originVal)
	tx, err = wToken.Approve(auth, rAddr, originVal)
	tx, err = btcToken.Approve(auth, rAddr, originVal)
	if err != nil {
		return err
	}

	// create pair
	tx, err = fToken.CreatePair(root, wethAddr, btcAddr)
	if err != nil {
		return err
	}

	// add liquidity, pool is 1:1
	liqVal := big.NewInt(1).Mul(big.NewInt(1e3), utils.Ether)
	tx, err = rToken.AddLiquidity(
		auth,
		wethAddr,
		btcAddr,
		liqVal,
		liqVal,
		big.NewInt(0),
		big.NewInt(0),
		auth.From,
		big.NewInt(2e9),
	)
	if err != nil {
		return err
	}

	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return err
	}

	// swap weth => btc
	swapVal := utils.Ether
	tx, err = rToken.SwapExactTokensForTokens(
		auth,
		swapVal,
		big.NewInt(0),
		[]common.Address{wethAddr, btcAddr},
		auth.From,
		big.NewInt(int64(header.Time)*2),
	)
	if err != nil {
		return err
	}
	_, err = bind.WaitMined(ctx, client, tx)
	return err
}
