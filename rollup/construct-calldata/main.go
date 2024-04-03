package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/database"
	"scroll-tech/common/types/encoding"
	"scroll-tech/common/types/encoding/codecv0"
	"scroll-tech/common/types/encoding/codecv1"
	"scroll-tech/rollup/internal/orm"
)

// ScrollChainMetaData contains all meta data concerning the ScrollChain contract.
var ScrollChainMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"batchIndex\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"batchHash\",\"type\":\"bytes32\"}],\"name\":\"CommitBatch\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"batchIndex\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"batchHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"stateRoot\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"withdrawRoot\",\"type\":\"bytes32\"}],\"name\":\"FinalizeBatch\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"batchIndex\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"batchHash\",\"type\":\"bytes32\"}],\"name\":\"RevertBatch\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"oldMaxNumTxInChunk\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"newMaxNumTxInChunk\",\"type\":\"uint256\"}],\"name\":\"UpdateMaxNumTxInChunk\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"status\",\"type\":\"bool\"}],\"name\":\"UpdateProver\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"status\",\"type\":\"bool\"}],\"name\":\"UpdateSequencer\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"},{\"internalType\":\"bytes\",\"name\":\"parentBatchHeader\",\"type\":\"bytes\"},{\"internalType\":\"bytes[]\",\"name\":\"chunks\",\"type\":\"bytes[]\"},{\"internalType\":\"bytes\",\"name\":\"skippedL1MessageBitmap\",\"type\":\"bytes\"}],\"name\":\"commitBatch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"batchIndex\",\"type\":\"uint256\"}],\"name\":\"committedBatches\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"batchHeader\",\"type\":\"bytes\"},{\"internalType\":\"bytes32\",\"name\":\"prevStateRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"postStateRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"withdrawRoot\",\"type\":\"bytes32\"}],\"name\":\"finalizeBatch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"batchHeader\",\"type\":\"bytes\"},{\"internalType\":\"bytes32\",\"name\":\"prevStateRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"postStateRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"withdrawRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"blobDataProof\",\"type\":\"bytes\"}],\"name\":\"finalizeBatch4844\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"batchHeader\",\"type\":\"bytes\"},{\"internalType\":\"bytes32\",\"name\":\"prevStateRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"postStateRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"withdrawRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"aggrProof\",\"type\":\"bytes\"}],\"name\":\"finalizeBatchWithProof\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"batchHeader\",\"type\":\"bytes\"},{\"internalType\":\"bytes32\",\"name\":\"prevStateRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"postStateRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"withdrawRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"blobDataProof\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"aggrProof\",\"type\":\"bytes\"}],\"name\":\"finalizeBatchWithProof4844\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"batchIndex\",\"type\":\"uint256\"}],\"name\":\"finalizedStateRoots\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_batchHeader\",\"type\":\"bytes\"},{\"internalType\":\"bytes32\",\"name\":\"_stateRoot\",\"type\":\"bytes32\"}],\"name\":\"importGenesisBatch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"batchIndex\",\"type\":\"uint256\"}],\"name\":\"isBatchFinalized\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"lastFinalizedBatchIndex\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"batchHeader\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"count\",\"type\":\"uint256\"}],\"name\":\"revertBatch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"batchIndex\",\"type\":\"uint256\"}],\"name\":\"withdrawRoots\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

func main() {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.LogfmtFormat()))
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)

	if len(os.Args) < 2 {
		log.Crit("no batch index provided")
		return
	}

	batchIndexStr := os.Args[1]
	batchIndexInt, err := strconv.Atoi(batchIndexStr)
	if err != nil || batchIndexInt <= 0 {
		log.Crit("invalid batch index", "indexStr", batchIndexStr, "err", err)
		return
	}
	batchIndex := uint64(batchIndexInt)

	db, err := database.InitDB(&database.Config{
		DriverName: "postgres",
		DSN:        os.Getenv("DB_DSN"),
		MaxOpenNum: 200,
		MaxIdleNum: 20,
	})
	if err != nil {
		log.Crit("failed to init db", "err", err)
	}
	defer func() {
		if deferErr := database.CloseDB(db); deferErr != nil {
			log.Error("failed to close db", "err", err)
		}
	}()

	l2BlockOrm := orm.NewL2Block(db)
	chunkOrm := orm.NewChunk(db)
	batchOrm := orm.NewBatch(db)

	fileName := "batches_calldata.txt"
	file, err := os.Create(fileName)
	if err != nil {
		log.Crit("failed to create file", "fileName", fileName, "err", err)
	}
	defer file.Close()

	for index := uint64(1); index <= batchIndex; index++ {
		dbBatch, err := batchOrm.GetBatchByIndex(context.Background(), index)
		if err != nil {
			log.Crit("failed to get batch", "index", index, "err", err)
			return
		}

		dbParentBatch, err := batchOrm.GetBatchByIndex(context.Background(), index-1)
		if err != nil {
			log.Crit("failed to get batch", "index", index-1, "err", err)
			return
		}

		dbChunks, err := chunkOrm.GetChunksInRange(context.Background(), dbBatch.StartChunkIndex, dbBatch.EndChunkIndex)
		if err != nil {
			log.Crit("failed to fetch chunks", "err", err)
			return
		}

		chunks := make([]*encoding.Chunk, len(dbChunks))
		for i, c := range dbChunks {
			blocks, err := l2BlockOrm.GetL2BlocksInRange(context.Background(), c.StartBlockNumber, c.EndBlockNumber)
			if err != nil {
				log.Crit("failed to fetch blocks", "err", err)
				return
			}
			chunks[i] = &encoding.Chunk{Blocks: blocks}
		}

		if index == 0 {
			calldata, err := constructCommitBatchPayloadCodecV0(dbBatch, dbParentBatch, dbChunks, chunks)
			if err != nil {
				log.Crit("fail to construct payload codecv0", "err", err)
			}

			_, err = file.WriteString(fmt.Sprintf("\nBatch Index: %d\n", index))
			if err != nil {
				log.Crit("failed to write batch index to file", "err", err)
			}

			_, err = file.WriteString("Calldata:\n")
			if err != nil {
				log.Crit("failed to write 'Calldata' label to file", "err", err)
			}

			_, err = file.WriteString(hex.EncodeToString(calldata) + "\n")
			if err != nil {
				log.Crit("failed to write calldata to file", "err", err)
			}
		} else {
			calldata, blob, err := constructCommitBatchPayloadCodecV1(dbBatch, dbParentBatch, dbChunks, chunks)
			if err != nil {
				log.Crit("fail to construct payload codecv1", "err", err)
			}

			_, err = file.WriteString(fmt.Sprintf("\nBatch Index: %d\n", index))
			if err != nil {
				log.Crit("failed to write batch index to file", "err", err)
			}

			_, err = file.WriteString("Calldata:\n")
			if err != nil {
				log.Crit("failed to write 'Calldata' label to file", "err", err)
			}

			_, err = file.WriteString(hex.EncodeToString(calldata) + "\n")
			if err != nil {
				log.Crit("failed to write calldata to file", "err", err)
			}

			//_, err = file.WriteString("Blob:\n")
			//if err != nil {
			//	log.Crit("failed to write 'Blob' label to file", "err", err)
			//}

			//_, err = file.WriteString(hex.EncodeToString(blob[:]) + "\n")
			//if err != nil {
			//	log.Crit("failed to write blob to file", "err", err)
			//}
		}
	}
}

func constructCommitBatchPayloadCodecV0(dbBatch *orm.Batch, dbParentBatch *orm.Batch, dbChunks []*orm.Chunk, chunks []*encoding.Chunk) ([]byte, error) {
	daBatch, err := codecv0.NewDABatchFromBytes(dbBatch.BatchHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to create DA batch from bytes: %w", err)
	}

	encodedChunks := make([][]byte, len(dbChunks))
	for i, c := range dbChunks {
		daChunk, createErr := codecv0.NewDAChunk(chunks[i], c.TotalL1MessagesPoppedBefore)
		if createErr != nil {
			return nil, fmt.Errorf("failed to create DA chunk: %w", createErr)
		}
		daChunkBytes, encodeErr := daChunk.Encode()
		if encodeErr != nil {
			return nil, fmt.Errorf("failed to encode DA chunk: %w", encodeErr)
		}
		encodedChunks[i] = daChunkBytes
	}

	l1RollupABI, err := ScrollChainMetaData.GetAbi()
	if err != nil {
		return nil, fmt.Errorf("failed to get abi: %w", err)
	}

	calldata, packErr := l1RollupABI.Pack("commitBatch", daBatch.Version, dbParentBatch.BatchHeader, encodedChunks, daBatch.SkippedL1MessageBitmap)
	if packErr != nil {
		return nil, fmt.Errorf("failed to pack commitBatch: %w", packErr)
	}
	return calldata, nil
}

func constructCommitBatchPayloadCodecV1(dbBatch *orm.Batch, dbParentBatch *orm.Batch, dbChunks []*orm.Chunk, chunks []*encoding.Chunk) ([]byte, *kzg4844.Blob, error) {
	batch := &encoding.Batch{
		Index:                      dbBatch.Index,
		TotalL1MessagePoppedBefore: dbChunks[0].TotalL1MessagesPoppedBefore,
		ParentBatchHash:            common.HexToHash(dbParentBatch.Hash),
		Chunks:                     chunks,
	}

	daBatch, createErr := codecv1.NewDABatch(batch)
	if createErr != nil {
		return nil, nil, fmt.Errorf("failed to create DA batch: %w", createErr)
	}

	encodedChunks := make([][]byte, len(dbChunks))
	for i, c := range dbChunks {
		daChunk, createErr := codecv1.NewDAChunk(chunks[i], c.TotalL1MessagesPoppedBefore)
		if createErr != nil {
			return nil, nil, fmt.Errorf("failed to create DA chunk: %w", createErr)
		}
		encodedChunks[i] = daChunk.Encode()
	}

	l1RollupABI, err := ScrollChainMetaData.GetAbi()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get abi: %w", err)
	}

	calldata, packErr := l1RollupABI.Pack("commitBatch", daBatch.Version, dbParentBatch.BatchHeader, encodedChunks, daBatch.SkippedL1MessageBitmap)
	if packErr != nil {
		return nil, nil, fmt.Errorf("failed to pack commitBatch: %w", packErr)
	}
	return calldata, daBatch.Blob(), nil
}
