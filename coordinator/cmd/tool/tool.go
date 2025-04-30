package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"scroll-tech/common/database"
	"scroll-tech/common/types/message"
	"scroll-tech/common/utils"
	"scroll-tech/common/version"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/orm"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

var app *cli.App

func init() {
	// Set up coordinator app info.
	app = cli.NewApp()
	app.Action = action
	app.Name = "coordinator-tool"
	app.Usage = "The Scroll L2 Coordinator Tool"
	app.Version = version.Version
	app.Flags = append(app.Flags, utils.CommonFlags...)
	app.Before = func(ctx *cli.Context) error {
		return utils.LogSetup(ctx)
	}
}

func action(ctx *cli.Context) error {
	cfgFile := ctx.String(utils.ConfigFileFlag.Name)
	cfg, err := config.NewConfig(cfgFile)
	if err != nil {
		log.Crit("failed to load config file", "config file", cfgFile, "error", err)
	}
	db, err := database.InitDB(cfg.DB)
	if err != nil {
		log.Crit("failed to init db connection", "err", err)
	}
	defer func() {
		if err = database.CloseDB(db); err != nil {
			log.Error("can not close db connection", "error", err)
		}
	}()

	batchOrm := orm.NewBatch(db)
	taskID := "9078d06d248f5ee7c910db6191809e4f4c9712ec236e27a5c03cfd50dfe69add"
	batches, err := batchOrm.GetBatchesByBundleHash(ctx.Context, taskID)
	if err != nil {
		log.Error("failed to get batch proofs for batch", "task_id", taskID, "error", err)
		return err
	}

	if len(batches) == 0 {
		log.Error("failed to get batch proofs for bundle, not found batch", "task_id", taskID)
		return fmt.Errorf("failed to get batch proofs for bundle task id:%s, no batch found", taskID)
	}

	hardForkName := "darwinV2"

	var batchProofs []message.BatchProof
	for _, batch := range batches {
		proof := message.NewBatchProof(hardForkName)
		if encodeErr := json.Unmarshal(batch.Proof, &proof); encodeErr != nil {
			log.Error("failed to unmarshal batch proof")
			return fmt.Errorf("failed to unmarshal proof: %w, bundle hash: %v, batch hash: %v", encodeErr, taskID, batch.Hash)
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

	parentBatch, err := batchOrm.GetBatchByHash(ctx, batches[0].ParentBatchHash)
	if err != nil {
		return fmt.Errorf("failed to get parent batch for batch task id:%s err:%w", taskID, err)
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
		return fmt.Errorf("failed to marshal batch proofs, taskID:%s err:%w", taskID, err)
	}

	taskMsg := &coordinatorType.GetTaskSchema{
		TaskID:       taskID,
		TaskType:     int(message.ProofTypeBundle),
		TaskData:     string(batchProofsBytes),
		HardForkName: hardForkName,
	}

	log.Info("task_msg", "data", taskMsg)
	return nil
}

func main() {
	// RunApp the coordinator.
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
