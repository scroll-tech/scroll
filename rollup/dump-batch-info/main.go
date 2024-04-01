package main

import (
	"context"
	"encoding/hex"
	"os"
	"strconv"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/database"
	"scroll-tech/common/types/encoding"
	"scroll-tech/common/types/encoding/codecv1"
	"scroll-tech/rollup/internal/orm"
)

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

	dbBatch, err := batchOrm.GetBatchByIndex(context.Background(), batchIndex)
	if err != nil {
		log.Crit("failed to get batch", "index", batchIndex, "err", err)
		return
	}

	dbParentBatch, err := batchOrm.GetBatchByIndex(context.Background(), batchIndex-1)
	if err != nil {
		log.Crit("failed to get batch", "index", batchIndex-1, "err", err)
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

	batch := &encoding.Batch{
		Index:                      dbBatch.Index,
		TotalL1MessagePoppedBefore: dbChunks[0].TotalL1MessagesPoppedBefore,
		ParentBatchHash:            common.HexToHash(dbParentBatch.Hash),
		Chunks:                     chunks,
	}

	daBatch, err := codecv1.NewDABatch(batch)
	if err != nil {
		log.Crit("failed to create DA batch", "err", err)
		return
	}

	blobDataProof, err := daBatch.BlobDataProof()
	if err != nil {
		log.Crit("failed to get blob data proof", "err", err)
		return
	}

	log.Info("batchMeta", "batchHash", daBatch.Hash().Hex(), "batchDataHash", daBatch.DataHash.Hex(), "blobDataProof", hex.EncodeToString(blobDataProof), "blobData", hex.EncodeToString(daBatch.Blob()[:]))
}
