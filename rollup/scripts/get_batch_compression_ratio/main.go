package main

/*
#cgo LDFLAGS: -lm -ldl -lscroll_zstd
#include <stdint.h>
char* compress_scroll_batch_bytes(uint8_t* src, uint64_t src_size, uint8_t* output_buf, uint64_t *output_buf_size);
*/
import "C"

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"unsafe"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/database"
	"scroll-tech/rollup/internal/orm"
)

func main() {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.LogfmtFormat()))
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)

	if len(os.Args) < 3 {
		log.Crit("Please provide start and end batch indices: ./script <start_index> <end_index>")
		return
	}

	startIndexStr := os.Args[1]
	endIndexStr := os.Args[2]

	startIndex, err := strconv.Atoi(startIndexStr)
	if err != nil || startIndex <= 0 {
		log.Crit("Invalid start batch index", "indexStr", startIndexStr, "err", err)
		return
	}

	endIndex, err := strconv.Atoi(endIndexStr)
	if err != nil || endIndex <= 0 {
		log.Crit("Invalid end batch index", "indexStr", endIndexStr, "err", err)
		return
	}

	if startIndex > endIndex {
		log.Crit("Start index must be less than or equal to end index")
		return
	}

	db, err := database.InitDB(&database.Config{
		DriverName: "postgres",
		DSN:        "postgres://postgres:scroll2022@localhost:7432/scroll",
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

	totalRawSize := uint64(0)
	totalCompressedSize := uint64(0)

	for i := startIndex; i <= endIndex; i++ {
		batchIndex := uint64(i)
		dbBatch, err := batchOrm.GetBatchByIndex(context.Background(), batchIndex)
		if err != nil {
			log.Crit("failed to get batch", "index", batchIndex, "err", err)
		}

		var dbParentBatch *orm.Batch
		if batchIndex >= 1 { // Skip fetching parent batch for the first batch
			dbParentBatch, err = batchOrm.GetBatchByIndex(context.Background(), batchIndex-1)
			if err != nil {
				log.Crit("failed to get parent batch", "index", batchIndex-1, "err", err)
			}
		}

		dbChunks, err := chunkOrm.GetChunksInRange(context.Background(), dbBatch.StartChunkIndex, dbBatch.EndChunkIndex)
		if err != nil {
			log.Crit("failed to fetch chunks", "err", err)
		}

		chunks := make([]*encoding.Chunk, len(dbChunks))
		for i, c := range dbChunks {
			blocks, err := l2BlockOrm.GetL2BlocksInRange(context.Background(), c.StartBlockNumber, c.EndBlockNumber)
			if err != nil {
				log.Crit("failed to fetch blocks", "err", err)
			}
			chunks[i] = &encoding.Chunk{Blocks: blocks}
		}

		batch := &encoding.Batch{
			Index:                      dbBatch.Index,
			TotalL1MessagePoppedBefore: dbChunks[0].TotalL1MessagesPoppedBefore,
			ParentBatchHash:            common.HexToHash(dbParentBatch.Hash),
			Chunks:                     chunks,
		}

		raw, compressed, err := estimateBatchL1CommitBlobSize(batch)
		if err != nil {
			log.Crit("failed to estimate batch l1 commit blob size", "err", err)
		}

		// compression_ratio = preimage_bytes / compressed_bytes
		log.Info("compression ratio", "raw length", raw, "compressed length", compressed, "ratio", 1.0*raw/compressed)

		totalRawSize += raw
		totalCompressedSize += compressed
	}

	batchCount := endIndex - startIndex + 1
	averageRawSize := float64(totalRawSize) / float64(batchCount)
	averageCompressedSize := float64(totalCompressedSize) / float64(batchCount)

	log.Info("Average compression ratio", "average raw length", averageRawSize, "average compressed length", averageCompressedSize, "ratio", averageRawSize/averageCompressedSize)
}

func estimateBatchL1CommitBlobSize(b *encoding.Batch) (uint64, uint64, error) {
	batchBytes, err := constructBatchPayload(b.Chunks)
	if err != nil {
		return 0, 0, err
	}
	blobBytes, err := compressScrollBatchBytes(batchBytes)
	if err != nil {
		return 0, 0, err
	}
	return uint64(len(batchBytes)), uint64(len(blobBytes)), nil
}

// constructBatchPayload constructs the batch payload.
// This function is only used in compressed batch payload length estimation.
func constructBatchPayload(chunks []*encoding.Chunk) ([]byte, error) {
	// metadata consists of num_chunks (2 bytes) and chunki_size (4 bytes per chunk)
	metadataLength := 2 + 45*4

	// the raw (un-compressed and un-padded) blob payload
	batchBytes := make([]byte, metadataLength)

	// batch metadata: num_chunks
	binary.BigEndian.PutUint16(batchBytes[0:], uint16(len(chunks)))

	// encode batch metadata and L2 transactions,
	for chunkID, chunk := range chunks {
		currentChunkStartIndex := len(batchBytes)

		for _, block := range chunk.Blocks {
			for _, tx := range block.Transactions {
				if tx.Type == types.L1MessageTxType {
					continue
				}

				// encode L2 txs into batch payload
				rlpTxData, err := encoding.ConvertTxDataToRLPEncoding(tx, false /* no mock */)
				if err != nil {
					return nil, err
				}
				batchBytes = append(batchBytes, rlpTxData...)
			}
		}

		// batch metadata: chunki_size
		if chunkSize := len(batchBytes) - currentChunkStartIndex; chunkSize != 0 {
			binary.BigEndian.PutUint32(batchBytes[2+4*chunkID:], uint32(chunkSize))
		}
	}
	return batchBytes, nil
}

// compressScrollBatchBytes compresses the given batch of bytes.
// The output buffer is allocated with an extra 128 bytes to accommodate metadata overhead or error message.
func compressScrollBatchBytes(batchBytes []byte) ([]byte, error) {
	srcSize := C.uint64_t(len(batchBytes))
	outbufSize := C.uint64_t(len(batchBytes) + 128) // Allocate output buffer with extra 128 bytes
	outbuf := make([]byte, outbufSize)

	if err := C.compress_scroll_batch_bytes((*C.uchar)(unsafe.Pointer(&batchBytes[0])), srcSize,
		(*C.uchar)(unsafe.Pointer(&outbuf[0])), &outbufSize); err != nil {
		return nil, fmt.Errorf("failed to compress scroll batch bytes: %s", C.GoString(err))
	}

	return outbuf[:int(outbufSize)], nil
}
