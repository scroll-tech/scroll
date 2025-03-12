package permissionless_batches

import (
	"context"
	"fmt"
	"math/big"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/da-codec/encoding"
	"gorm.io/gorm"

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rpc"
	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
	bridgeAbi "scroll-tech/rollup/abi"
	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/controller/sender"
	"scroll-tech/rollup/internal/orm"
)

type Submitter struct {
	ctx context.Context

	db         *gorm.DB
	l2BlockOrm *orm.L2Block
	chunkOrm   *orm.Chunk
	batchOrm   *orm.Batch
	bundleOrm  *orm.Bundle

	cfg *config.RelayerConfig

	finalizeSender *sender.Sender
	l1RollupABI    *abi.ABI

	chainCfg *params.ChainConfig
}

func NewSubmitter(ctx context.Context, db *gorm.DB, cfg *config.RelayerConfig, chainCfg *params.ChainConfig) (*Submitter, error) {
	registry := prometheus.DefaultRegisterer
	finalizeSender, err := sender.NewSender(ctx, cfg.SenderConfig, cfg.FinalizeSenderSignerConfig, "permissionless_batches_submitter", "finalize_sender", types.SenderTypeFinalizeBatch, db, registry)
	if err != nil {
		return nil, fmt.Errorf("new finalize sender failed, err: %w", err)
	}

	return &Submitter{
		ctx:            ctx,
		db:             db,
		l2BlockOrm:     orm.NewL2Block(db),
		chunkOrm:       orm.NewChunk(db),
		batchOrm:       orm.NewBatch(db),
		bundleOrm:      orm.NewBundle(db),
		cfg:            cfg,
		finalizeSender: finalizeSender,
		l1RollupABI:    bridgeAbi.ScrollChainABI,
		chainCfg:       chainCfg,
	}, nil

}

func (s *Submitter) Submit(withProof bool) error {
	// Check if the bundle is already finalized
	bundle, err := s.bundleOrm.GetLatestBundle(s.ctx)
	if err != nil {
		return fmt.Errorf("error loading latest bundle: %w", err)
	}

	if bundle.Index != defaultFakeRestoredBundleIndex+1 {
		return fmt.Errorf("unexpected bundle index %d with hash %s, expected %d", bundle.Index, bundle.Hash, defaultFakeRestoredBundleIndex+1)
	}

	if types.RollupStatus(bundle.RollupStatus) == types.RollupFinalized {
		return fmt.Errorf("bundle %d %s is already finalized. nothing to do", bundle.Index, bundle.Hash)
	}

	if bundle.StartBatchIndex != bundle.EndBatchIndex {
		return fmt.Errorf("bundle %d %s has unexpected batch indices (should only contain a single batch): start %d, end %d", bundle.Index, bundle.Hash, bundle.StartBatchIndex, bundle.EndBatchIndex)
	}
	if bundle.StartBatchHash != bundle.EndBatchHash {
		return fmt.Errorf("bundle %d %s has unexpected batch hashes (should only contain a single batch): start %s, end %s", bundle.Index, bundle.Hash, bundle.StartBatchHash, bundle.EndBatchHash)
	}

	batch, err := s.batchOrm.GetBatchByIndex(s.ctx, bundle.StartBatchIndex)
	if err != nil {
		return fmt.Errorf("failed to load batch %d: %w", bundle.StartBatchIndex, err)
	}
	if batch == nil {
		return fmt.Errorf("batch %d not found", bundle.StartBatchIndex)
	}
	if batch.Hash != bundle.StartBatchHash {
		return fmt.Errorf("bundle %d %s has unexpected batch hash: %s", bundle.Index, bundle.Hash, batch.Hash)
	}

	log.Info("submitting batch", "index", batch.Index, "hash", batch.Hash)

	endChunk, err := s.chunkOrm.GetChunkByIndex(s.ctx, batch.EndChunkIndex)
	if err != nil || endChunk == nil {
		return fmt.Errorf("failed to get end chunk with index %d of batch: %w", batch.EndChunkIndex, err)
	}

	var aggProof message.BundleProof
	if withProof {
		firstChunk, err := s.chunkOrm.GetChunkByIndex(s.ctx, batch.StartChunkIndex)
		if err != nil || firstChunk == nil {
			return fmt.Errorf("failed to get first chunk %d of batch: %w", batch.StartChunkIndex, err)
		}

		hardForkName := encoding.GetHardforkName(s.chainCfg, firstChunk.StartBlockNumber, firstChunk.StartBlockTime)

		aggProof, err = s.bundleOrm.GetVerifiedProofByHash(s.ctx, bundle.Hash, hardForkName)
		if err != nil {
			return fmt.Errorf("failed to get verified proof by bundle index: %d, err: %w", bundle.Index, err)
		}

		if err = aggProof.SanityCheck(); err != nil {
			return fmt.Errorf("failed to check agg_proof sanity, index: %d, err: %w", bundle.Index, err)
		}
	}

	var calldata []byte
	var blob *kzg4844.Blob
	switch encoding.CodecVersion(bundle.CodecVersion) {
	case encoding.CodecV7:
		calldata, blob, err = s.constructCommitAndFinalizeCalldataAndBlob(batch, endChunk, aggProof)
		if err != nil {
			return fmt.Errorf("failed to construct CommitAndFinalize calldata and blob, bundle index: %v, batch index: %v, err: %w", bundle.Index, batch.Index, err)
		}
	default:
		return fmt.Errorf("unsupported codec version in finalizeBundle, bundle index: %v, version: %d", bundle.Index, bundle.CodecVersion)
	}
	//
	fmt.Println(len(blob))
	txHash, err := s.finalizeSender.SendTransaction("commitAndFinalize-"+bundle.Hash, &s.cfg.RollupContractAddress, calldata, []*kzg4844.Blob{blob}, 0)
	if err != nil {
		//fmt.Println("blob", common.Bytes2Hex(blob[:]))
		log.Error("commitAndFinalize in layer1 failed", "with proof", withProof, "index", bundle.Index,
			"batch index", bundle.StartBatchIndex,
			"RollupContractAddress", s.cfg.RollupContractAddress, "err", err, "calldata", common.Bytes2Hex(calldata))
		if typedErr, ok := err.(rpc.Error); ok {
			fmt.Println("errorData", typedErr.ErrorCode())
		}
		return fmt.Errorf("commitAndFinalize failed, bundle index: %d, err: %w", bundle.Index, err)
	}

	log.Info("commitAndFinalize in layer1", "with proof", withProof, "batch index", bundle.StartBatchIndex, "tx hash", txHash.String())

	// Updating rollup status in database.
	err = s.db.Transaction(func(dbTX *gorm.DB) error {
		if err = s.batchOrm.UpdateFinalizeTxHashAndRollupStatusByBundleHash(s.ctx, bundle.Hash, txHash.String(), types.RollupFinalizing, dbTX); err != nil {
			log.Warn("UpdateFinalizeTxHashAndRollupStatusByBundleHash failed", "index", bundle.Index, "bundle hash", bundle.Hash, "tx hash", txHash.String(), "err", err)
			return err
		}

		if err = s.bundleOrm.UpdateFinalizeTxHashAndRollupStatus(s.ctx, bundle.Hash, txHash.String(), types.RollupFinalizing, dbTX); err != nil {
			log.Warn("UpdateFinalizeTxHashAndRollupStatus failed", "index", bundle.Index, "bundle hash", bundle.Hash, "tx hash", txHash.String(), "err", err)
			return err
		}

		return nil
	})
	if err != nil {
		log.Warn("failed to update rollup status of bundle and batches", "err", err)
		return err
	}

	// Updating the proving status when finalizing without proof, thus the coordinator could omit this task.
	// it isn't a necessary step, so don't put in a transaction with UpdateFinalizeTxHashAndRollupStatus
	if !withProof {
		txErr := s.db.Transaction(func(dbTX *gorm.DB) error {
			if updateErr := s.bundleOrm.UpdateProvingStatus(s.ctx, bundle.Hash, types.ProvingTaskVerified, dbTX); updateErr != nil {
				return updateErr
			}
			if updateErr := s.batchOrm.UpdateProvingStatusByBundleHash(s.ctx, bundle.Hash, types.ProvingTaskVerified, dbTX); updateErr != nil {
				return updateErr
			}
			for batchIndex := bundle.StartBatchIndex; batchIndex <= bundle.EndBatchIndex; batchIndex++ {
				tmpBatch, getErr := s.batchOrm.GetBatchByIndex(s.ctx, batchIndex)
				if getErr != nil {
					return getErr
				}
				if updateErr := s.chunkOrm.UpdateProvingStatusByBatchHash(s.ctx, tmpBatch.Hash, types.ProvingTaskVerified, dbTX); updateErr != nil {
					return updateErr
				}
			}
			return nil
		})
		if txErr != nil {
			log.Error("Updating chunk and batch proving status when finalizing without proof failure", "bundleHash", bundle.Hash, "err", txErr)
		}
	}

	return nil
}

func (s *Submitter) constructCommitAndFinalizeCalldataAndBlob(batch *orm.Batch, endChunk *orm.Chunk, aggProof message.BundleProof) ([]byte, *kzg4844.Blob, error) {
	// Create the FinalizeStruct tuple as an abi-compatible struct
	finalizeStruct := struct {
		BatchHeader                  []byte
		TotalL1MessagesPoppedOverall *big.Int
		PostStateRoot                common.Hash
		WithdrawRoot                 common.Hash
		ZkProof                      []byte
	}{
		BatchHeader:                  batch.BatchHeader,
		TotalL1MessagesPoppedOverall: new(big.Int).SetUint64(endChunk.TotalL1MessagesPoppedBefore + endChunk.TotalL1MessagesPoppedInChunk),
		PostStateRoot:                common.HexToHash(batch.StateRoot),
		WithdrawRoot:                 common.HexToHash(batch.WithdrawRoot),
	}
	if aggProof != nil {
		finalizeStruct.ZkProof = aggProof.Proof()
	}

	calldata, err := s.l1RollupABI.Pack("commitAndFinalizeBatch", uint8(batch.CodecVersion), common.HexToHash(batch.ParentBatchHash), finalizeStruct)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to pack commitAndFinalizeBatch: %w", err)
	}

	chunks, err := s.chunkOrm.GetChunksInRange(s.ctx, batch.StartChunkIndex, batch.EndChunkIndex)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get chunks in range for batch %d: %w", batch.Index, err)
	}
	if chunks[len(chunks)-1].Index != batch.EndChunkIndex {
		return nil, nil, fmt.Errorf("unexpected last chunk index %d, expected %d", chunks[len(chunks)-1].Index, batch.EndChunkIndex)
	}

	var batchBlocks []*encoding.Block
	for _, c := range chunks {
		blocks, err := s.l2BlockOrm.GetL2BlocksInRange(s.ctx, c.StartBlockNumber, c.EndBlockNumber)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get blocks in range for batch %d: %w", batch.Index, err)
		}

		batchBlocks = append(batchBlocks, blocks...)
	}

	encodingBatch := &encoding.Batch{
		Index:                  batch.Index,
		ParentBatchHash:        common.HexToHash(batch.ParentBatchHash),
		PrevL1MessageQueueHash: common.HexToHash(batch.PrevL1MessageQueueHash),
		PostL1MessageQueueHash: common.HexToHash(batch.PostL1MessageQueueHash),
		Blocks:                 batchBlocks,
	}

	codec, err := encoding.CodecFromVersion(encoding.CodecVersion(batch.CodecVersion))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get codec from version %d, err: %w", batch.CodecVersion, err)
	}

	daBatch, err := codec.NewDABatch(encodingBatch)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create DA batch: %w", err)
	}

	return calldata, daBatch.Blob(), nil
}
