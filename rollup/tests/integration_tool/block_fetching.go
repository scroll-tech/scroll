package main

import (
	"context"
	"fmt"
	"math/big"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"
)

func fetchAndStoreBlocks(ctx context.Context, from, to uint64) ([]*encoding.Block, error) {
	validiumMode := cfg.ValidiumMode
	cfg := cfg.FetchConfig
	client, err := rpc.Dial(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to connect l2 geth, endpoint %s, err %v", cfg.Endpoint, err)
	}
	defer client.Close()

	ethCli := ethclient.NewClient(client)
	var blocks []*encoding.Block
	for number := from; number <= to; number++ {
		log.Debug("retrieving block", "height", number)
		block, err := ethCli.BlockByNumber(ctx, new(big.Int).SetUint64(number))
		if err != nil {
			return nil, fmt.Errorf("failed to BlockByNumber: %v. number: %v", err, number)
		}

		blockTxs := block.Transactions()

		var count int
		for _, tx := range blockTxs {
			if tx.IsL1MessageTx() {
				count++
			}
		}
		log.Info("retrieved block", "height", block.Header().Number, "hash", block.Header().Hash().String(), "L1 message count", count)

		// use original (encrypted) L1 message txs in validium mode
		if validiumMode {
			var txs []*types.Transaction

			if count > 0 {
				log.Info("Fetching encrypted messages in validium mode")
				err = client.CallContext(ctx, &txs, "scroll_getL1MessagesInBlock", block.Hash(), "synced")
				if err != nil {
					return nil, fmt.Errorf("failed to get L1 messages: %v, block hash: %v", err, block.Hash().Hex())
				}
			}

			// sanity check
			if len(txs) != count {
				return nil, fmt.Errorf("L1 message count mismatch: expected %d, got %d", count, len(txs))
			}

			for ii := 0; ii < count; ii++ {
				// sanity check
				if blockTxs[ii].AsL1MessageTx().QueueIndex != txs[ii].AsL1MessageTx().QueueIndex {
					return nil, fmt.Errorf("L1 message queue index mismatch at index %d: expected %d, got %d", ii, blockTxs[ii].AsL1MessageTx().QueueIndex, txs[ii].AsL1MessageTx().QueueIndex)
				}

				log.Info("Replacing L1 message tx in validium mode", "index", ii, "queueIndex", txs[ii].AsL1MessageTx().QueueIndex, "decryptedTxHash", blockTxs[ii].Hash().Hex(), "originalTxHash", txs[ii].Hash().Hex())
				blockTxs[ii] = txs[ii]
			}
		}

		withdrawRoot, err3 := ethCli.StorageAt(ctx, cfg.L2MessageQueueAddress, cfg.WithdrawTrieRootSlot, big.NewInt(int64(number)))
		if err3 != nil {
			return nil, fmt.Errorf("failed to get withdrawRoot: %v. number: %v", err3, number)
		}
		blocks = append(blocks, &encoding.Block{
			Header:       block.Header(),
			Transactions: encoding.TxsToTxsData(blockTxs),
			WithdrawRoot: common.BytesToHash(withdrawRoot),
		})
	}

	return blocks, nil
}
