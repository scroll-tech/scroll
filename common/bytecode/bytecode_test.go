package bytecode

import (
	"context"
	"encoding/json"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
	"math/big"
	"os"
	"scroll-tech/common/bytecode/L2/predeploys"
	"scroll-tech/common/bytecode/erc20"
	"scroll-tech/common/docker"
	"testing"
)

var (
	l2gethImg docker.ImgInstance
	client    *ethclient.Client
	root      *bind.TransactOpts
	auth0     *bind.TransactOpts
)

func setup(t *testing.T) {
	l2gethImg = docker.NewTestL2Docker(t)
	client, _ = ethclient.Dial(l2gethImg.Endpoint())
	chainID, _ := client.ChainID(context.Background())

	priv, err := crypto.HexToECDSA("1212121212121212121212121212121212121212121212121212121212121212")
	assert.NoError(t, err)
	root, err = bind.NewKeyedTransactorWithChainID(priv, chainID)
	assert.NoError(t, err)

	sk, err := crypto.GenerateKey()
	assert.NoError(t, err)
	auth0, err = bind.NewKeyedTransactorWithChainID(sk, chainID)
	assert.NoError(t, err)
}

func getGenssis(file string) (*core.Genesis, error) {
	var (
		genesis core.Genesis
	)
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &genesis)
	if err != nil {
		return nil, err
	}
	return &genesis, nil
}

func setGenesis(file string, genesis *core.Genesis) error {
	data, err := json.MarshalIndent(&genesis, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(file, data, 0644)
}

func resetAllocInGenesis(client *ethclient.Client, contract common.Address, number *big.Int) (*core.GenesisAccount, error) {
	code, err := client.CodeAt(context.Background(), contract, number)
	if err != nil {
		return nil, err
	}
	trace, err := client.GetBlockTraceByNumber(context.Background(), number)
	if err != nil {
		return nil, err
	}

	var sstore = make(map[common.Hash]common.Hash)
	for _, tx := range trace.ExecutionResults {
		for i := len(tx.StructLogs) - 1; i >= 0; i-- {
			log := tx.StructLogs[i]
			if log.Op == "SSTORE" {
				for k, v := range log.Storage {
					sstore[common.HexToHash(k)] = common.HexToHash(v)
				}
			}
		}
	}

	return &core.GenesisAccount{
		Balance: big.NewInt(0),
		Code:    code,
		Storage: sstore,
	}, nil
}

func TestDeployL2(t *testing.T) {
	setup(t)

	file := "../docker/l2geth/genesis.json"
	genesis, err := getGenssis(file)
	assert.NoError(t, err)

	/*t.Run("testDeployWhiteList", func(t *testing.T) {
		testDeployWhiteList(t, &genesis.Alloc)
	})*/
	t.Run("testERC20", func(t *testing.T) {
		testERC20(t, &genesis.Alloc)
	})
	assert.NoError(t, setGenesis(file, genesis))

	t.Cleanup(func() {
		_ = l2gethImg.Stop()
	})
}

func testDeployWhiteList(t *testing.T, alloc *core.GenesisAlloc) {
	addr, tx, token, err := predeploys.DeployWhitelist(root, client, root.From)
	assert.NoError(t, err)
	_, err = bind.WaitDeployed(context.Background(), client, tx)
	assert.NoError(t, err)

	tx, err = token.TransferOwnership(root, common.HexToAddress("21cdbd4361a5944e4be5b08723ecc5e2e38a9841"))
	receipt, err := bind.WaitMined(context.Background(), client, tx)
	assert.NoError(t, err)

	account, err := resetAllocInGenesis(client, addr, receipt.BlockNumber)
	assert.NoError(t, err)
	(*alloc)[addr] = *account
}

func testERC20(t *testing.T, alloc *core.GenesisAlloc) {
	bls, ok := big.NewInt(0).SetString("20000000000000000000000000000000000000000000000", 10)
	assert.Equal(t, true, ok)
	addr, tx, token, err := erc20.DeployERC20Mock(root, client, "ETH", "ETH coin", root.From, bls)
	assert.NoError(t, err)
	_, err = bind.WaitDeployed(context.Background(), client, tx)

	tx, err = token.Transfer(root, common.HexToAddress("21cdbd4361a5944e4be5b08723ecc5e2e38a9841"), big.NewInt(1000))
	assert.NoError(t, err)
	receipt, err := bind.WaitMined(context.Background(), client, tx)
	assert.NoError(t, err)

	account, err := resetAllocInGenesis(client, addr, receipt.BlockNumber)
	assert.NoError(t, err)
	(*alloc)[addr] = *account
}

func TestWhiteList(t *testing.T) {
	cli, err := ethclient.Dial("ws://127.0.0.1:8546")
	assert.NoError(t, err)
	token, err := predeploys.NewWhitelist(common.HexToAddress("21cdbd4361a5944e4be5b08723ecc5e2e38a9841"), cli)
	assert.NoError(t, err)

	owner, err := token.Owner(nil)
	assert.NoError(t, err)
	t.Logf(owner.String())
	assert.Equal(t, "21cdbd4361a5944e4be5b08723ecc5e2e38a9841", owner.String())
}

func TestVerifyERC20(t *testing.T) {
	cli, err := ethclient.Dial("ws://127.0.0.1:8546")
	assert.NoError(t, err)
	token, err := erc20.NewERC20Mock(common.HexToAddress("21cdbd4361a5944e4be5b08723ecc5e2e38a9841"), cli)
	assert.NoError(t, err)

	bls, err := token.BalanceOf(nil, common.HexToAddress("1c5a77d9fa7ef466951b2f01f724bca3a5820b63"))
	assert.NoError(t, err)
	t.Logf(bls.String())

	bls, err = token.BalanceOf(nil, common.HexToAddress("21cdbd4361a5944e4be5b08723ecc5e2e38a9841"))
	assert.NoError(t, err)
	t.Logf(bls.String())
}
