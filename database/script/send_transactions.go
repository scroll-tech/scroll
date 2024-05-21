package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
)

var chainID = new(big.Int).SetUint64(222222)

func main() {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.LogfmtFormat()))
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)

	privateKey, err := crypto.HexToECDSA(os.Getenv("L2_DEPLOYER_PRIVATE_KEY"))
	if err != nil {
		log.Crit("failed to create private key", "err", err)
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Crit("failed to cast public key to ECDSA")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	client, err := ethclient.Dial(os.Getenv("SCROLL_L2_DEPLOYMENT_RPC"))
	if err != nil {
		log.Crit("failed to connect to network", "err", err)
	}

	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Crit("failed to get pending nonce", "err", err)
	}

	contractAddress := common.HexToAddress(os.Getenv("L2_TEST_CURIE_OPCODES_ADDR"))

	contractABI, err := bind.BindABI(common.ReadFile("abi.json"))
	if err != nil {
		log.Crit("failed to bind ABI", "err", err)
	}

	// 创建合约实例
	contract := bind.NewBoundContract(contractAddress, contractABI, client, client)

	// 创建 transactor
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		log.Crit("failed to create transactor", "chainID", chainID, "err", err)
	}
	auth.Nonce = new(big.Int).SetUint64(nonce)
	auth.Value = big.NewInt(0)
	auth.GasLimit = uint64(300000)
	auth.GasPrice = new(big.Int).SetUint64(1000000000) // 1 Gwei

	for i := 0; i < 1000; i++ {
		// useTloadTstore
		tx, err := contract.Transact(auth, "useTloadTstore", new(big.Int).SetUint64(9876543210))
		if err != nil {
			log.Error("failed to send useTloadTstore transaction", "err", err)
		} else {
			fmt.Printf("Sent useTloadTstore transaction with nonce: %d, tx hash: %s\n", auth.Nonce.Uint64(), tx.Hash().String())
		}
		auth.Nonce.Add(auth.Nonce, big.NewInt(1))

		// useMcopy
		tx, err = contract.Transact(auth, "useMcopy")
		if err != nil {
			log.Error("failed to send useMcopy transaction", "err", err)
		} else {
			fmt.Printf("Sent useMcopy transaction with nonce: %d, tx hash: %s\n", auth.Nonce.Uint64(), tx.Hash().String())
		}
		auth.Nonce.Add(auth.Nonce, big.NewInt(1))

		// useBaseFee
		tx, err = contract.Transact(auth, "useBaseFee")
		if err != nil {
			log.Error("failed to send useBaseFee transaction", "err", err)
		} else {
			fmt.Printf("Sent useBaseFee transaction with nonce: %d, tx hash: %s\n", auth.Nonce.Uint64(), tx.Hash().String())
		}
		auth.Nonce.Add(auth.Nonce, big.NewInt(1))
	}

	fmt.Println("Done")
}
