package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
)

func main() {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.LogfmtFormat()))
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)

	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(os.Getenv("L2_DEPLOYER_PRIVATE_KEY"), "0x"))
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

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, new(big.Int).SetUint64(222222))
	if err != nil {
		log.Crit("failed to initialize keyed transactor with chain ID", "err", err)
	}

	abiJSON, err := os.ReadFile("abi.json")
	if err != nil {
		log.Crit("failed to read ABI file", "err", err)
	}

	l2TestCurieOpcodesMetaData := &bind.MetaData{ABI: string(abiJSON)}
	l2TestCurieOpcodesAbi, err := l2TestCurieOpcodesMetaData.GetAbi()
	if err != nil {
		log.Crit("failed to get abi", "err", err)
	}

	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Crit("failed to get pending nonce", "err", err)
	}

	useTloadTstoreCalldata, err := l2TestCurieOpcodesAbi.Pack("useTloadTstore", new(big.Int).SetUint64(9876543210))
	if err != nil {
		log.Crit("failed to pack useTloadTstore calldata", "err", err)
	}

	useMcopyCalldata, err := l2TestCurieOpcodesAbi.Pack("useMcopy")
	if err != nil {
		log.Crit("failed to pack useMcopy calldata", "err", err)
	}

	useBaseFee, err := l2TestCurieOpcodesAbi.Pack("useBaseFee")
	if err != nil {
		log.Crit("failed to pack useBaseFee calldata", "err", err)
	}

	l2TestCurieOpcodesAddr := common.HexToAddress(os.Getenv("L2_TEST_CURIE_OPCODES_ADDR"))

	txTypes := []int{
		LegacyTxType,
		AccessListTxType,
		DynamicFeeTxType,
	}

	accessLists := []types.AccessList{
		nil,
		{
			{Address: common.HexToAddress("0x0000000000000000000000000000000000000000"), StorageKeys: []common.Hash{
				common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
			}},
		},
		{
			{Address: common.HexToAddress("0x1000000000000000000000000000000000000000"), StorageKeys: []common.Hash{
				common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001")}},
		},
		{
			{Address: common.HexToAddress("0x2000000000000000000000000000000000000000"), StorageKeys: []common.Hash{
				common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000002"),
				common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000003"),
			}},
			{Address: common.HexToAddress("0x3000000000000000000000000000000000000000"), StorageKeys: []common.Hash{
				common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000004"),
			}},
		},
		{
			{Address: common.HexToAddress("0x4000000000000000000000000000000000000000"), StorageKeys: []common.Hash{
				common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000005"),
				common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000005"), // repetitive storage key
			}},
		},
	}

	for i := 0; i < 1000; i++ {
		for _, txType := range txTypes {
			for _, accessList := range accessLists {
				if err := sendTransaction(client, auth, txType, &l2TestCurieOpcodesAddr, nonce, accessList, nil, useTloadTstoreCalldata); err != nil {
					log.Crit("failed to send transaction", "nonce", nonce, "err", err)
				}
				nonce += 1

				if err := sendTransaction(client, auth, txType, &l2TestCurieOpcodesAddr, nonce, accessList, nil, useMcopyCalldata); err != nil {
					log.Crit("failed to send transaction", "nonce", nonce, "err", err)
				}
				nonce += 1

				if err := sendTransaction(client, auth, txType, &l2TestCurieOpcodesAddr, nonce, accessList, nil, useBaseFee); err != nil {
					log.Crit("failed to send transaction", "nonce", nonce, "err", err)
				}
				nonce += 1

				if err := sendTransaction(client, auth, txType, &fromAddress, nonce, accessList, nil, []byte{0x01, 0x02, 0x03, 0x04}); err != nil {
					log.Crit("failed to send transaction", "nonce", nonce, "err", err)
				}
				nonce += 1

				if err := sendTransaction(client, auth, txType, &fromAddress, nonce, accessList, new(big.Int).SetUint64(1), []byte{0x01, 0x02, 0x03, 0x04}); err != nil {
					log.Crit("failed to send transaction", "nonce", nonce, "err", err)
				}
				nonce += 1

				if err := sendTransaction(client, auth, txType, &fromAddress, nonce, accessList, new(big.Int).SetUint64(1), nil); err != nil {
					log.Crit("failed to send transaction", "nonce", nonce, "err", err)
				}
				nonce += 1
			}
		}
	}
}

const (
	LegacyTxType     = 1
	AccessListTxType = 2
	DynamicFeeTxType = 3
)

func sendTransaction(client *ethclient.Client, auth *bind.TransactOpts, txType int, to *common.Address, nonce uint64, accessList types.AccessList, value *big.Int, data []byte) error {
	var txData types.TxData
	switch txType {
	case LegacyTxType:
		txData = &types.LegacyTx{
			Nonce:    nonce,
			GasPrice: new(big.Int).SetUint64(1000000000),
			Gas:      300000,
			To:       to,
			Value:    value,
			Data:     data,
		}
	case AccessListTxType:
		txData = &types.AccessListTx{
			ChainID:    new(big.Int).SetUint64(222222),
			Nonce:      nonce,
			GasPrice:   new(big.Int).SetUint64(1000000000),
			Gas:        300000,
			To:         to,
			Value:      value,
			Data:       data,
			AccessList: accessList,
		}
	case DynamicFeeTxType:
		txData = &types.DynamicFeeTx{
			ChainID:    new(big.Int).SetUint64(222222),
			Nonce:      nonce,
			GasTipCap:  new(big.Int).SetUint64(1000000000),
			GasFeeCap:  new(big.Int).SetUint64(1000000000),
			Gas:        300000,
			To:         to,
			Value:      value,
			Data:       data,
			AccessList: accessList,
		}
	default:
		return fmt.Errorf("invalid transaction type: %d", txType)
	}

	signedTx, err := auth.Signer(auth.From, types.NewTx(txData))
	if err != nil {
		return fmt.Errorf("failed to sign tx: %w", err)
	}

	if err = client.SendTransaction(context.Background(), signedTx); err != nil {
		return fmt.Errorf("failed to send tx: %w", err)
	}

	log.Info("transaction sent", "txHash", signedTx.Hash().Hex())

	var receipt *types.Receipt
	for {
		receipt, err = client.TransactionReceipt(context.Background(), signedTx.Hash())
		if err == nil {
			if receipt.Status != types.ReceiptStatusSuccessful {
				return fmt.Errorf("transaction failed: %s", signedTx.Hash().Hex())
			}
			break
		}
		log.Warn("waiting for receipt", "txHash", signedTx.Hash())
		time.Sleep(2 * time.Second)
	}

	log.Info("Sent transaction", "txHash", signedTx.Hash().Hex(), "from", auth.From.Hex(), "nonce", signedTx.Nonce(), "to", to.Hex())
	return nil
}
