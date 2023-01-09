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
	l1 "scroll-tech/common/bytecode/L1"
	"scroll-tech/common/bytecode/L2/predeploys"
	"scroll-tech/common/docker"
	"testing"
)

var (
	l2gethImg docker.ImgInstance
	client    *ethclient.Client
	root      *bind.TransactOpts
)

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

func setSenesis(file string, genesis *core.Genesis, alloc *core.GenesisAlloc) error {
	data, err := json.Marshal(alloc)
	if err != nil {
		return err
	}
	err = genesis.Alloc.UnmarshalJSON(data)
	if err != nil {
		return err
	}

	data, err = json.MarshalIndent(&genesis, "", "    ")
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
				break
			}
		}
	}

	return &core.GenesisAccount{
		Balance: big.NewInt(0),
		Code:    code,
		Storage: sstore,
	}, nil
}

func setup(t *testing.T) {
	l2gethImg = docker.NewTestL2Docker(t)
	client, _ = ethclient.Dial(l2gethImg.Endpoint())
	chainID, _ := client.ChainID(context.Background())

	priv, err := crypto.HexToECDSA("1212121212121212121212121212121212121212121212121212121212121212")
	assert.NoError(t, err)
	root, err = bind.NewKeyedTransactorWithChainID(priv, chainID)
	assert.NoError(t, err)
}

func TestDeployL2(t *testing.T) {
	setup(t)

	file := "../docker/l2geth/genesis.json"
	genesis, err := getGenssis(file)
	assert.NoError(t, err)

	var (
		alloc = core.GenesisAlloc{
			common.HexToAddress("1c5a77d9fa7ef466951b2f01f724bca3a5820b63"): core.GenesisAccount{
				Balance: big.NewInt(0).SetBytes(common.FromHex("0x200000000000000000000000000000000000000000000000000000000000000")),
			},
		}
	)

	//t.Run("testDeployL1", func(t *testing.T) {
	//	testDeployL1(t, &alloc)
	//})
	t.Run("testDeployWhiteList", func(t *testing.T) {
		testDeployWhiteList(t, &alloc)
	})
	assert.NoError(t, setSenesis(file, genesis, &alloc))

	t.Cleanup(func() {
		_ = l2gethImg.Stop()
	})
}

func testDeployL1(t *testing.T, alloc *core.GenesisAlloc) {
	// Deploy L1ScrollMessenger.
	l1Scroll, tx, _, err := l1.DeployL1ScrollMessenger(root, client)
	assert.NoError(t, err)
	receipt, err := bind.WaitMined(context.Background(), client, tx)
	assert.NoError(t, err)

	account, err := resetAllocInGenesis(client, l1Scroll, receipt.BlockNumber)
	assert.NoError(t, err)
	(*alloc)[l1Scroll] = *account

	addr, tx, _, err := predeploys.DeployL2ToL1MessagePasser(root, client, l1Scroll)
	receipt, err = bind.WaitMined(context.Background(), client, tx)

	account, err = resetAllocInGenesis(client, addr, receipt.BlockNumber)
	assert.NoError(t, err)
	(*alloc)[addr] = *account
}

func testDeployWhiteList(t *testing.T, alloc *core.GenesisAlloc) {
	addr, tx, _, err := predeploys.DeployWhitelist(root, client, root.From)
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
	assert.Equal(t, root.From.String(), owner.String())
}
