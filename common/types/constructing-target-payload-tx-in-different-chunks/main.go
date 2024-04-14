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

const targetTxSize = 126914

func main() {
	privateKeyHex := "0000000000000000000000000000000000000000000000000000000000000042"
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		log.Fatalf("Invalid private key: %v", err)
	}

	client, err := ethclient.Dial("http://localhost:9999")
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(222222))
	if err != nil {
		log.Fatalf("Failed to create transactor with chain ID 222222: %v", err)
	}

	nonce, err := client.PendingNonceAt(context.Background(), auth.From)
	if err != nil {
		log.Fatalf("Failed to retrieve account nonce: %v", err)
	}

	totalTxNum := []uint64{2, 3, 4, 5, 6}
	for _, num := range totalTxNum {
		prepareAndSendTransactions(client, auth, nonce, num)
		nonce += num
	}
}

func prepareAndSendTransactions(client *ethclient.Client, auth *bind.TransactOpts, initialNonce uint64, totalTxNum uint64) error {
	gasLimit := uint64(5000000)
	gasPrice := big.NewInt(1000000000)

	var signedTxs []*types.Transaction
	payloadSum := 0

	dataPayload := make([]byte, targetTxSize/totalTxNum)
	for i := range dataPayload {
		dataPayload[i] = 0xff
	}

	for i := uint64(0); i < totalTxNum-1; i++ {
		txData := &types.LegacyTx{
			Nonce:    initialNonce + i,
			GasPrice: gasPrice,
			Gas:      gasLimit,
			To:       &auth.From,
			Data:     dataPayload,
		}

		signedTx, err := auth.Signer(auth.From, types.NewTx(txData))
		if err != nil {
			log.Fatalf("Failed to sign tx: %v", err)
		}

		rlpTxData, err := rlp.EncodeToBytes(signedTx)
		if err != nil {
			log.Fatalf("Failed to RLP encode the tx: %v", err)
		}

		payloadSum += len(rlpTxData)
		signedTxs = append(signedTxs, signedTx)
	}

	fmt.Println("payload sum", payloadSum)

	lowerBound := 0
	upperBound := targetTxSize
	for lowerBound <= upperBound {
		mid := (lowerBound + upperBound) / 2
		data := make([]byte, mid)
		for i := range data {
			data[i] = 0xff
		}

		txData := &types.LegacyTx{
			Nonce:    initialNonce + totalTxNum - 1,
			GasPrice: gasPrice,
			Gas:      gasLimit,
			To:       &auth.From,
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

		if payloadSum+txSize < targetTxSize {
			lowerBound = mid + 1
		} else if payloadSum+txSize > targetTxSize {
			upperBound = mid - 1
		} else {
			fmt.Println("payloadSum+txSize", payloadSum+txSize)
			signedTxs = append(signedTxs, signedTx)
			break
		}
	}

	for _, signedTx := range signedTxs {
		if err := client.SendTransaction(context.Background(), signedTx); err != nil {
			return fmt.Errorf("failed to send transaction: %v", err)
		}
		fmt.Printf("Transaction with nonce %d sent\n", signedTx.Nonce())
		time.Sleep(10 * time.Second)
	}

	return nil
}
