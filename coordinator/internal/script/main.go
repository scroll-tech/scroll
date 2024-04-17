package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/database"
	"scroll-tech/common/types/message"
	"scroll-tech/coordinator/internal/orm"
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

	for i := startIndex; i <= endIndex; i++ {
		batchIndex := uint64(i)
		resultBytes, err := getBatchTask(db, batchIndex)
		if err != nil {
			log.Crit("failed to get batch task", "batchIndex", batchIndex, "err", err)
			continue
		}

		outputFilename := fmt.Sprintf("batch_task_%d.json", batchIndex)
		if err = os.WriteFile(outputFilename, resultBytes, 0644); err != nil {
			log.Crit("failed to write output file", "filename", outputFilename, "err", err)
		}
	}
}

func getBatchTask(db *gorm.DB, batchIndex uint64) ([]byte, error) {
	batchHash, err := orm.NewBatch(db).GetBatchHashByIndex(context.Background(), batchIndex)
	if err != nil {
		err = fmt.Errorf("failed to get batch hash by index: %d err: %w ", batchIndex, err)
		return nil, err
	}

	chunks, err := orm.NewChunk(db).GetChunksByBatchHash(context.Background(), batchHash)
	if err != nil {
		err = fmt.Errorf("failed to get chunk proofs for batch task id: %s err: %w ", batchHash, err)
		return nil, err
	}

	var chunkProofs []*message.ChunkProof
	var chunkInfos []*message.ChunkInfo
	for _, chunk := range chunks {
		var proof message.ChunkProof
		if encodeErr := json.Unmarshal(chunk.Proof, &proof); encodeErr != nil {
			return nil, fmt.Errorf("Chunk.GetProofsByBatchHash unmarshal proof error: %w, batch hash: %v, chunk hash: %v", encodeErr, batchHash, chunk.Hash)
		}
		chunkProofs = append(chunkProofs, &proof)

		chunkInfo := message.ChunkInfo{
			ChainID:       534351, // sepolia
			PrevStateRoot: common.HexToHash(chunk.ParentChunkStateRoot),
			PostStateRoot: common.HexToHash(chunk.StateRoot),
			WithdrawRoot:  common.HexToHash(chunk.WithdrawRoot),
			DataHash:      common.HexToHash(chunk.Hash),
			IsPadding:     false,
		}
		if proof.ChunkInfo != nil {
			chunkInfo.TxBytes = proof.ChunkInfo.TxBytes
		}
		chunkInfos = append(chunkInfos, &chunkInfo)
	}

	taskDetail := message.BatchTaskDetail{
		ChunkInfos:  chunkInfos,
		ChunkProofs: chunkProofs,
	}

	chunkProofsBytes, err := json.MarshalIndent(taskDetail, "", "    ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal chunk proofs, taskID:%s err:%w", batchHash, err)
	}

	return chunkProofsBytes, nil
}
