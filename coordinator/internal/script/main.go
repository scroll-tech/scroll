package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/scroll-tech/da-codec/encoding"
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

	if len(os.Args) < 3 {
		log.Crit("Usage: go run main.go <batch|bundle> <args>")
		return
	}

	command := os.Args[1]
	arg := os.Args[2]

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
			log.Error("failed to close db", "err", deferErr)
		}
	}()

	switch command {
	case "batch":
		handleBatchCommand(db, arg)
	case "bundle":
		handleBundleCommand(db, arg)
	default:
		log.Crit("unknown command", "command", command)
	}
}

func handleBatchCommand(db *gorm.DB, indexRange string) {
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

func handleBundleCommand(db *gorm.DB, bundleHash string) {
	resultBytes, err := getBundleTaskDetail(db, bundleHash)
	if err != nil {
		log.Crit("failed to get bundle task detail", "bundleHash", bundleHash, "err", err)
		return
	}

	outputFilename := fmt.Sprintf("bundle_task_%s.json", bundleHash)
	if err = os.WriteFile(outputFilename, resultBytes, 0644); err != nil {
		log.Crit("failed to write output file", "filename", outputFilename, "err", err)
	}
}

func getBundleTaskDetail(db *gorm.DB, bundleHash string) ([]byte, error) {
	bundleProof, err := orm.NewBundle(db).GetBundleByHash()
	if err != nil {
		return nil, fmt.Errorf("failed to get bundle proof by hash %s: %w", bundleHash, err)
	}

	bundleProof.Proof
	batches, err := orm.NewBatch(db).GetBatchesByBundleHash(context.Background(), bundleHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get batches by bundle hash %s: %w", bundleHash, err)
	}

	if len(batches) == 0 {
		return nil, fmt.Errorf("no batch found for bundle hash %s", bundleHash)
	}

	var batchProofs []message.BatchProof
	for _, batch := range batches {
		proof := message.NewBatchProof("euclid")
		if encodeErr := json.Unmarshal(batch.Proof, &proof); encodeErr != nil {
			return nil, fmt.Errorf("failed to unmarshal batch proof: %w, bundle hash: %v, batch hash: %v", encodeErr, bundleHash, batch.Hash)
		}
		batchProofs = append(batchProofs, proof)
	}

	taskDetail := message.BundleTaskDetail{
		BatchProofs: batchProofs,
	}

	return json.MarshalIndent(taskDetail, "", "    ")
}

func getBatchTask(db *gorm.DB, batchIndex uint64) ([]byte, error) {
	batch, err := orm.NewBatch(db).GetBatchByIndex(context.Background(), batchIndex)
	if err != nil {
		err = fmt.Errorf("failed to get batch hash by index: %d err: %w ", batchIndex, err)
		return nil, err
	}

	chunks, err := orm.NewChunk(db).GetChunksByBatchHash(context.Background(), batch.Hash)
	if err != nil {
		err = fmt.Errorf("failed to get chunk proofs for batch task id: %s err: %w ", batch.Hash, err)
		return nil, err
	}

	var chunkProofs []message.ChunkProof
	var chunkInfos []*message.ChunkInfo
	for _, chunk := range chunks {
		proof := message.NewChunkProof("euclid")
		if encodeErr := json.Unmarshal(chunk.Proof, &proof); encodeErr != nil {
			return nil, fmt.Errorf("Chunk.GetProofsByBatchHash unmarshal proof error: %w, batch hash: %v, chunk hash: %v", encodeErr, batch.Hash, chunk.Hash)
		}
		chunkProofs = append(chunkProofs, proof)

		chunkInfo := message.ChunkInfo{
			ChainID:       534352,
			PrevStateRoot: common.HexToHash(chunk.ParentChunkStateRoot),
			PostStateRoot: common.HexToHash(chunk.StateRoot),
			WithdrawRoot:  common.HexToHash(chunk.WithdrawRoot),
			DataHash:      common.HexToHash(chunk.Hash),
			IsPadding:     false,
		}
		if haloProot, ok := proof.(*message.Halo2ChunkProof); ok {
			if haloProot.ChunkInfo != nil {
				chunkInfo.TxBytes = haloProot.ChunkInfo.TxBytes
			}
		}
		chunkInfos = append(chunkInfos, &chunkInfo)
	}

	taskDetail, err := getBatchTaskDetail(batch, chunkInfos, chunkProofs)
	if err != nil {
		return nil, fmt.Errorf("failed to get batch task detail, taskID:%s err:%w", batch.Hash, err)
	}

	chunkProofsBytes, err := json.MarshalIndent(taskDetail, "", "    ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal chunk proofs, taskID:%s err:%w", batch.Hash, err)
	}

	return chunkProofsBytes, nil
}

func getBatchTaskDetail(dbBatch *orm.Batch, chunkInfos []*message.ChunkInfo, chunkProofs []message.ChunkProof) (*message.BatchTaskDetail, error) {
	taskDetail := &message.BatchTaskDetail{
		ChunkInfos:  chunkInfos,
		ChunkProofs: chunkProofs,
	}

	dbBatchCodecVersion := encoding.CodecVersion(dbBatch.CodecVersion)
	switch dbBatchCodecVersion {
	case encoding.CodecV3, encoding.CodecV4, encoding.CodecV6:
	default:
		return taskDetail, nil
	}

	codec, err := encoding.CodecFromVersion(encoding.CodecVersion(dbBatch.CodecVersion))
	if err != nil {
		return nil, fmt.Errorf("failed to get codec from version %d, err: %w", dbBatch.CodecVersion, err)
	}

	batchHeader, decodeErr := codec.NewDABatchFromBytes(dbBatch.BatchHeader)
	if decodeErr != nil {
		return nil, fmt.Errorf("failed to decode batch header version %d: %w", dbBatch.CodecVersion, decodeErr)
	}
	taskDetail.BatchHeader = batchHeader
	taskDetail.BlobBytes = dbBatch.BlobBytes

	return taskDetail, nil
}
