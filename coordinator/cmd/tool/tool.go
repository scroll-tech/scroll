package main

import (
	"encoding/json"
	"fmt"
	"os"

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
	taskID := "fa9a290c8f1a46dc626fa67d626fadfe4803968ce776383996f3ae12504a2591"
	batches, err := batchOrm.GetBatchesByBundleHash(ctx.Context, taskID)
	if err != nil {
		log.Error("failed to get batch proofs for batch", "task_id", taskID, "error", err)
		return err
	}

	if len(batches) == 0 {
		log.Error("failed to get batch proofs for bundle, not found batch", "task_id", taskID)
		return fmt.Errorf("failed to get batch proofs for bundle task id:%s, no batch found", taskID)
	}

	var batchProofs []*message.BatchProof
	for _, batch := range batches {
		var proof message.BatchProof
		if encodeErr := json.Unmarshal(batch.Proof, &proof); encodeErr != nil {
			log.Error("failed to unmarshal batch proof")
			return fmt.Errorf("failed to unmarshal proof: %w, bundle hash: %v, batch hash: %v", encodeErr, taskID, batch.Hash)
		}
		batchProofs = append(batchProofs, &proof)
	}

	taskDetail := message.BundleTaskDetail{
		BatchProofs: batchProofs,
	}

	batchProofsBytes, err := json.Marshal(taskDetail)
	if err != nil {
		log.Error("failed to marshal batch proof")
		return fmt.Errorf("failed to marshal batch proofs, taskID:%s err:%w", taskID, err)
	}

	taskMsg := &coordinatorType.GetTaskSchema{
		TaskID:   taskID,
		TaskType: int(message.ProofTypeBundle),
		TaskData: string(batchProofsBytes),
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
