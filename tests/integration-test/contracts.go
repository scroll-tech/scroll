package integration

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math"
	"math/big"
	"math/rand"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"

	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/integration-test/abi/dao"
	"scroll-tech/integration-test/abi/erc20"
	"scroll-tech/integration-test/abi/greeter"
	"scroll-tech/integration-test/abi/nft"
	"scroll-tech/integration-test/abi/sushi"
	"scroll-tech/integration-test/abi/uniswap/factory"
	"scroll-tech/integration-test/abi/uniswap/router"
	"scroll-tech/integration-test/abi/uniswap/weth9"
	"scroll-tech/integration-test/abi/vote"

	bridgeConfig "scroll-tech/bridge/config"
	"scroll-tech/bridge/sender"

	"scroll-tech/common/utils"
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
	_, err = newSender.SendTransaction("native_01", &to, value, nil)
	<-newSender.ConfirmChan()
	return err
}

func newERC20(ctx context.Context, client *ethclient.Client, root, auth *bind.TransactOpts) error {
	_, tx, erc20Token, err := erc20.DeployERC20Mock(root, client, "WETH coin", "WETH", root.From, big.NewInt(1e4))
	if err != nil {
		return err
	}
	_, _ = bind.WaitMined(ctx, client, tx)

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
	_, err = bind.WaitMined(ctx, client, tx)

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
		return fmt.Errorf("err from deploy: %v", err)
	}

	_, err = bind.WaitMined(ctx, client, tx)

	tokenId := big.NewInt(rand.Int63())
	tx, err = token.Mint(root, root.From, tokenId)
	if err != nil {
		return fmt.Errorf("err from mint: %v", err)
	}
	_, err = bind.WaitMined(ctx, client, tx)

	tx, err = token.TransferFrom(root, root.From, auth.From, tokenId)
	if err != nil {
		return fmt.Errorf("err from transfer: %v", err)
	}
	_, err = bind.WaitMined(ctx, client, tx)

	tokenId = big.NewInt(rand.Int63())
	tx, err = token.Mint(root, root.From, tokenId)
	if err != nil {
		return fmt.Errorf("err from mint: %v", err)
	}

	_, err = bind.WaitMined(ctx, client, tx)
	tx, err = token.Burn(root, tokenId)
	if err != nil {
		return fmt.Errorf("err from burn: %v", err)
	}
	_, err = bind.WaitMined(ctx, client, tx)
	return err
}

func newSushi(ctx context.Context, client *ethclient.Client, root *bind.TransactOpts) error {
	sushiAddr, tx, sushiToken, err := sushi.DeploySushiToken(root, client)
	if err != nil {
		return err
	}

	_, err = bind.WaitMined(ctx, client, tx)
	chefAddr, tx, chefToken, err := sushi.DeployMasterChef(root, client, sushiAddr, root.From, big.NewInt(1), big.NewInt(1), big.NewInt(math.MaxInt))
	if err != nil {
		return err
	}
	_, err = bind.WaitMined(ctx, client, tx)

	amount := big.NewInt(1e18)
	tx, err = sushiToken.Mint(root, root.From, amount)
	if err != nil {
		return err
	}
	_, err = bind.WaitMined(ctx, client, tx)

	allocPoint := utils.Ether
	tx, err = chefToken.Add(root, allocPoint, sushiAddr, true)
	if err != nil {
		return err
	}
	_, err = bind.WaitMined(ctx, client, tx)

	pid, err := chefToken.PoolLength(&bind.CallOpts{Pending: true})
	if err != nil {
		return err
	}
	pid.Sub(pid, big.NewInt(1))
	tx, err = chefToken.Set(root, pid, allocPoint, true)
	if err != nil {
		return err
	}
	_, err = bind.WaitMined(ctx, client, tx)

	tx, err = sushiToken.Approve(root, chefAddr, amount)
	if err != nil {
		return err
	}
	_, err = bind.WaitMined(ctx, client, tx)

	// deposit amount to chef
	tx, err = chefToken.Deposit(root, pid, amount)
	if err != nil {
		return err
	}
	_, err = bind.WaitMined(ctx, client, tx)

	// change sushiToken's owner to masterChef.
	tx, err = sushiToken.TransferOwnership(root, chefAddr)
	if err != nil {
		return err
	}
	_, err = bind.WaitMined(ctx, client, tx)

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
	_, err = bind.WaitMined(ctx, client, tx)

	_, tx, daoToken, err := dao.DeployGovernorMock(root, client, "governor mock", voteAddr, big.NewInt(1), big.NewInt(1), big.NewInt(100))
	if err != nil {
		return err
	}
	_, err = bind.WaitMined(ctx, client, tx)

	callData := [][]byte{big.NewInt(1).Bytes()}
	target := common.BigToAddress(big.NewInt(1))
	value := big.NewInt(1)
	description := "dao propose test"
	tx, err = daoToken.Propose(root, []common.Address{target}, []*big.Int{value}, callData, description)
	if err != nil {
		return err
	}
	_, err = bind.WaitMined(ctx, client, tx)

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
		return fmt.Errorf("err from deployweth9: %v", err)
	}
	_, err = bind.WaitMined(ctx, client, tx)

	// deploy factory
	fAddr, tx, fToken, err := factory.DeployUniswapV2Factory(root, client, root.From)
	if err != nil {
		return fmt.Errorf("err from deployuniswapv2Fact: %v", err)
	}
	_, err = bind.WaitMined(ctx, client, tx)

	// deploy router
	rAddr, tx, rToken, err := router.DeployUniswapV2Router02(root, client, fAddr, wethAddr)
	if err != nil {
		return fmt.Errorf("err from deployuniswapRouter: %v", err)
	}
	_, err = bind.WaitMined(ctx, client, tx)

	originVal := big.NewInt(1).Mul(big.NewInt(3e3), utils.Ether)
	btcAddr, tx, btcToken, err := erc20.DeployERC20Mock(root, client, "BTC coin", "BTC", auth.From, originVal)
	if err != nil {
		return fmt.Errorf("err from ERC2OMock: %v", err)
	}
	_, err = bind.WaitMined(ctx, client, tx)

	// init balance
	auth.GasPrice = big.NewInt(1108583800)
	auth.GasLimit = 11529000
	out, _ := client.BalanceAt(ctx, auth.From, nil)
	log.Warn(fmt.Sprintf("balance at auth: %d", out.Int64()))
	tx, err = wToken.Deposit(auth)
	tx, err = wToken.Approve(auth, rAddr, originVal)
	tx, err = btcToken.Approve(auth, rAddr, originVal)
	if err != nil {
		return fmt.Errorf("err from initbalance: %v", err)
	}

	// create pair
	tx, err = fToken.CreatePair(root, wethAddr, btcAddr)
	if err != nil {
		return err
	}
	_, err = bind.WaitMined(ctx, client, tx)

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
	_, err = bind.WaitMined(ctx, client, tx)

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
