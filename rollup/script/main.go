package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/da-codec/encoding/codecv3"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/database"
	"scroll-tech/common/types/message"
	"scroll-tech/rollup/internal/orm"
)

func main() {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.LogfmtFormat()))
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)

	if len(os.Args) < 2 {
		log.Crit("no batch index range provided")
		return
	}

	indexRange := os.Args[1]
	indices := strings.Split(indexRange, "-")
	if len(indices) != 2 {
		log.Crit("invalid batch index range format. Use start-end", "providedRange", indexRange)
		return
	}

	startIndex, err := strconv.Atoi(indices[0])
	endIndex, err2 := strconv.Atoi(indices[1])
	if err != nil || err2 != nil || startIndex > endIndex {
		log.Crit("invalid batch index range", "start", indices[0], "end", indices[1], "err", err, "err2", err2)
		return
	}

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

	dbParentBatch, getErr := orm.NewBatch(db).GetBatchByIndex(context.Background(), uint64(startIndex)-1)
	if getErr != nil {
		log.Crit("failed to get parent batch header", "err", getErr)
	}
	parentBatchHash := common.HexToHash(dbParentBatch.Hash)

	for i := startIndex; i <= endIndex; i++ {
		batchIndex := uint64(i)
		resultBytes, currentBatchHash, err := getBatchTask(db, parentBatchHash, batchIndex)
		if err != nil {
			log.Crit("failed to get batch task", "batchIndex", batchIndex, "err", err)
			continue
		}

		parentBatchHash = currentBatchHash

		outputFilename := fmt.Sprintf("batch_task_%d.json", batchIndex)
		if err = os.WriteFile(outputFilename, resultBytes, 0644); err != nil {
			log.Crit("failed to write output file", "filename", outputFilename, "err", err)
		}
	}
}

func getBatchTask(db *gorm.DB, parentBatchHash common.Hash, batchIndex uint64) ([]byte, common.Hash, error) {
	dbBatch, err := orm.NewBatch(db).GetBatchByIndex(context.Background(), batchIndex)
	if err != nil {
		err = fmt.Errorf("failed to get batch hash by index: %d, err: %w ", batchIndex, err)
		return nil, common.Hash{}, err
	}

	dbChunks, err := orm.NewChunk(db).GetChunksInRange(context.Background(), dbBatch.StartChunkIndex, dbBatch.EndChunkIndex)
	if err != nil {
		err = fmt.Errorf("failed to get chunk proofs for batch hash: %s, err: %w ", dbBatch.Hash, err)
		return nil, common.Hash{}, err
	}

	var chunkProofs []*message.ChunkProof
	var chunkInfos []*message.ChunkInfo
	for _, chunk := range dbChunks {
		var proof message.ChunkProof
		if encodeErr := json.Unmarshal(chunk.Proof, &proof); encodeErr != nil {
			return nil, common.Hash{}, fmt.Errorf("unmarshal proof error: %w, batch hash: %v, chunk hash: %v", encodeErr, dbBatch.Hash, chunk.Hash)
		}
		chunkProofs = append(chunkProofs, &proof)
		chunkInfos = append(chunkInfos, proof.ChunkInfo)
	}

	taskDetail := message.BatchTaskDetail{
		ChunkInfos:  chunkInfos,
		ChunkProofs: chunkProofs,
	}

	chunks := make([]*encoding.Chunk, len(dbChunks))
	for i, c := range dbChunks {
		blocks, getErr := orm.NewL2Block(db).GetL2BlocksInRange(context.Background(), c.StartBlockNumber, c.EndBlockNumber)
		if getErr != nil {
			log.Error("failed to get blocks in range", "err", getErr)
			return nil, common.Hash{}, getErr
		}
		chunks[i] = &encoding.Chunk{Blocks: blocks}
	}

	batch := &encoding.Batch{
		Index:                      dbBatch.Index,
		TotalL1MessagePoppedBefore: dbChunks[0].TotalL1MessagesPoppedBefore,
		ParentBatchHash:            parentBatchHash,
		Chunks:                     chunks,
	}

	daBatch, createErr := codecv3.NewDABatch(batch)
	if createErr != nil {
		return nil, common.Hash{}, fmt.Errorf("failed to create DA batch: %w", createErr)
	}

	taskDetail.BatchHeader = daBatch

	jsonData, err := json.MarshalIndent(taskDetail.BatchHeader, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return nil, common.Hash{}, err
	}
	fmt.Println(string(jsonData))

	chunkProofsBytes, err := json.MarshalIndent(taskDetail, "", "    ")
	if err != nil {
		return nil, common.Hash{}, fmt.Errorf("failed to marshal chunk proofs, batch hash: %s, err: %w", dbBatch.Hash, err)
	}

	return chunkProofsBytes, daBatch.Hash(), nil
}
