package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/rlp"
)

const targetTxSize = 126976

func main() {
	privateKeyHex := "0000000000000000000000000000000000000000000000000000000000000042"
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		log.Fatalf("Invalid private key: %v", err)
	}

	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	client, err := ethclient.Dial("http://localhost:8545")
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}

	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatalf("Failed to retrieve account nonce: %v", err)
	}

	fmt.Println("nonce", nonce)

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(222222))
	if err != nil {
		log.Fatalf("Failed to create transactor with chain ID 222222: %v", err)
	}

	gasLimit := uint64(3000000)
	gasPrice := big.NewInt(2000000000)

	lowerBound := 0
	upperBound := targetTxSize

	for lowerBound <= upperBound {
		mid := (lowerBound + upperBound) / 2
		data := make([]byte, mid)

		txData := &types.LegacyTx{
			Nonce:    nonce,
			GasPrice: gasPrice,
			Gas:      gasLimit,
			To:       &fromAddress,
			Data:     data,
		}

		signedTx, err := auth.Signer(auth.From, types.NewTx(txData))
		if err != nil {
			log.Fatalf("Failed to sign tx: %v", err)
		}

		rlpTxData, err := rlp.EncodeToBytes(signedTx)
		if err != nil {
			log.Fatalf("Failed to RLP encode the tx: %v", err)
		}
		txSize := len(rlpTxData)

		if txSize < targetTxSize {
			lowerBound = mid + 1
		} else if txSize > targetTxSize {
			upperBound = mid - 1
		} else {
			fmt.Printf("Found correct payload size: %d bytes\n", mid)
			fmt.Printf("RLP encoded transaction size: %d bytes\n", len(rlpTxData))
			fmt.Printf("Transaction hash: %s\n", signedTx.Hash().Hex())

			err = client.SendTransaction(context.Background(), signedTx)
			if err != nil {
				log.Fatalf("Failed to send transaction: %v", err)
			}
			fmt.Printf("Transaction sent! Hash: %s\n", signedTx.Hash().Hex())

			fmt.Println("Polling for transaction receipt...")
			for {
				receipt, err := client.TransactionReceipt(context.Background(), signedTx.Hash())
				if err == nil {
					fmt.Printf("Transaction receipt received: status %v\n", receipt.Status)
					return
				}
				if err.Error() != "not found" {
					log.Fatalf("Failed to get transaction receipt: %v", err)
					break
				}
				fmt.Println("Transaction receipt not found yet. Waiting for mining...")
				time.Sleep(2 * time.Second)
			}
			return
		}
	}

	fmt.Println("Could not find the exact payload size for 128 KiB RLP encoded transaction.")
}
