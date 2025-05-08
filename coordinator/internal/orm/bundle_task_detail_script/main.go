package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/database"
	"scroll-tech/common/types/message"
	"scroll-tech/coordinator/internal/orm"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

func main() {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.LogfmtFormat()))
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)

	dbDSN := os.Getenv("DB_DSN")
	if dbDSN == "" {
		log.Crit("DB_DSN environment variable is not set")
		return
	}

	db, err := database.InitDB(&database.Config{
		DriverName: "postgres",
		DSN:        dbDSN,
		MaxOpenNum: 200,
		MaxIdleNum: 20,
	})
	if err != nil {
		log.Crit("failed to init db connection", "err", err)
		return
	}
	defer func() {
		if err = database.CloseDB(db); err != nil {
			log.Error("can not close db connection", "error", err)
		}
	}()

	taskID := "7591df9c01efb7efc25359cee4600edce7e307f4d8806837a05cf510067a2819"
	hardForkName := "euclidV2"

	log.Info("Processing bundle", "taskID", taskID, "hardForkName", hardForkName)

	batchOrm := orm.NewBatch(db)
	batches, err := batchOrm.GetBatchesByBundleHash(context.Background(), taskID)
	if err != nil {
		log.Error("failed to get batch proofs for batch", "task_id", taskID, "error", err)
		os.Exit(1)
	}

	if len(batches) == 0 {
		log.Error("failed to get batch proofs for bundle, not found batch", "task_id", taskID)
		os.Exit(1)
	}

	var batchProofs []message.BatchProof
	for _, batch := range batches {
		proof := message.NewBatchProof(hardForkName)
		if encodeErr := json.Unmarshal(batch.Proof, &proof); encodeErr != nil {
			log.Error("failed to unmarshal batch proof", "error", encodeErr)
			os.Exit(1)
		}
		batchProofs = append(batchProofs, proof)
	}

	taskDetail := message.BundleTaskDetail{
		BatchProofs: batchProofs,
	}

	if hardForkName == message.EuclidV2Fork {
		taskDetail.ForkName = message.EuclidV2ForkNameForProver
	} else if hardForkName == message.EuclidFork {
		taskDetail.ForkName = message.EuclidForkNameForProver
	}

	parentBatch, err := batchOrm.GetBatchByHash(context.Background(), batches[0].ParentBatchHash)
	if err != nil {
		log.Error("failed to get parent batch", "task_id", taskID, "error", err)
		os.Exit(1)
	}

	taskDetail.BundleInfo = &message.OpenVMBundleInfo{
		ChainID:       534352,
		PrevStateRoot: common.HexToHash(parentBatch.StateRoot),
		PostStateRoot: common.HexToHash(batches[len(batches)-1].StateRoot),
		WithdrawRoot:  common.HexToHash(batches[len(batches)-1].WithdrawRoot),
		NumBatches:    uint32(len(batches)),
		PrevBatchHash: common.HexToHash(batches[0].ParentBatchHash),
		BatchHash:     common.HexToHash(batches[len(batches)-1].Hash),
	}

	if hardForkName == message.EuclidV2Fork {
		taskDetail.BundleInfo.MsgQueueHash = common.HexToHash(batches[len(batches)-1].PostL1MessageQueueHash)
	}

	batchProofsBytes, err := json.Marshal(taskDetail)
	if err != nil {
		log.Error("failed to marshal batch proofs", "task_id", taskID, "error", err)
		os.Exit(1)
	}

	taskMsg := &coordinatorType.GetTaskSchema{
		TaskID:       taskID,
		TaskType:     int(message.ProofTypeBundle),
		TaskData:     string(batchProofsBytes),
		HardForkName: hardForkName,
	}

	outputFilename := fmt.Sprintf("bundle_task_%s.json", taskID)
	if err = os.WriteFile(outputFilename, batchProofsBytes, 0644); err != nil {
		log.Error("failed to write output file", "filename", outputFilename, "error", err)
		os.Exit(1)
	}

	log.Info("Task details saved to file", "filename", outputFilename)
	log.Info("Task message", "data", taskMsg)
}
