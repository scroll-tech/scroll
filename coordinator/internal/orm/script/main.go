package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
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
		fmt.Println("chunk index: ", chunk.Index)
		fmt.Print("chunk proof: ", chunk.Proof)
		proof := message.NewChunkProof("euclid")
		if encodeErr := json.Unmarshal(chunk.Proof, &proof); encodeErr != nil {
			return nil, fmt.Errorf("Chunk.GetProofsByBatchHash unmarshal proof error: %w, batch hash: %v, chunk hash: %v", encodeErr, batch.Hash, chunk.Hash)
		}
		chunkProofs = append(chunkProofs, proof)

		chunkInfo := message.ChunkInfo{
			ChainID:          534351,
			PrevStateRoot:    common.HexToHash(chunk.ParentChunkStateRoot),
			PostStateRoot:    common.HexToHash(chunk.StateRoot),
			WithdrawRoot:     common.HexToHash(chunk.WithdrawRoot),
			DataHash:         common.HexToHash(chunk.Hash),
			PrevMsgQueueHash: common.HexToHash(chunk.PrevL1MessageQueueHash),
			PostMsgQueueHash: common.HexToHash(chunk.PostL1MessageQueueHash),
			IsPadding:        false,
		}
		if openvmProof, ok := proof.(*message.OpenVMChunkProof); ok {
			chunkInfo.InitialBlockNumber = openvmProof.MetaData.ChunkInfo.InitialBlockNumber
			chunkInfo.BlockCtxs = openvmProof.MetaData.ChunkInfo.BlockCtxs
			chunkInfo.TxDataLength = openvmProof.MetaData.ChunkInfo.TxDataLength
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

	if dbBatchCodecVersion >= encoding.CodecV7 {
		taskDetail.ForkName = message.EuclidV2ForkNameForProver
	} else {
		taskDetail.ForkName = message.EuclidForkNameForProver
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

	challengeDigest, kzgCommitment, kzgProof, err := codec.BlobDataProofFromBlobBytes(dbBatch.BlobBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to get challenge digest from blob bytes, taskID: %s, err: %w", dbBatch.Hash, err)
	}

	taskDetail.ChallengeDigest = challengeDigest
	taskDetail.KzgProof = message.Byte48{Big: hexutil.Big(*new(big.Int).SetBytes(kzgProof[:]))}
	taskDetail.KzgCommitment = message.Byte48{Big: hexutil.Big(*new(big.Int).SetBytes(kzgCommitment[:]))}

	return taskDetail, nil
}
