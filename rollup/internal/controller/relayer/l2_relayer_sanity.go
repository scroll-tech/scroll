package relayer

import (
	"fmt"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/rollup/internal/orm"
)

// validateMessageQueueConsistency validates L1 message queue hash consistency
func (r *Layer2Relayer) validateMessageQueueConsistency(batchIndex uint64, chunks []*orm.Chunk, prevL1MsgQueueHash common.Hash, postL1MsgQueueHash common.Hash) error {
	if batchIndex == 0 {
		return nil
	}

	if len(chunks) == 0 {
		return fmt.Errorf("batch %d has no chunks for message queue validation", batchIndex)
	}

	firstChunk := chunks[0]
	lastChunk := chunks[len(chunks)-1]

	// Calculate total L1 messages in this batch
	var totalL1MessagesInBatch uint64
	for _, chunk := range chunks {
		totalL1MessagesInBatch += chunk.TotalL1MessagesPoppedInChunk
	}

	// If there were L1 messages processed before this batch, prev hash should not be zero
	if firstChunk.TotalL1MessagesPoppedBefore > 0 && prevL1MsgQueueHash == (common.Hash{}) {
		return fmt.Errorf("batch %d prev L1 message queue hash is zero but %d L1 messages were processed before", batchIndex, firstChunk.TotalL1MessagesPoppedBefore)
	}

	// If there are any L1 messages processed up to this batch, post hash should not be zero
	totalL1MessagesProcessed := lastChunk.TotalL1MessagesPoppedBefore + lastChunk.TotalL1MessagesPoppedInChunk
	if totalL1MessagesProcessed > 0 && postL1MsgQueueHash == (common.Hash{}) {
		return fmt.Errorf("batch %d post L1 message queue hash is zero but %d L1 messages were processed in total", batchIndex, totalL1MessagesProcessed)
	}

	// Prev and post queue hashes should be different if L1 messages were processed in this batch
	if totalL1MessagesInBatch > 0 && prevL1MsgQueueHash == postL1MsgQueueHash {
		return fmt.Errorf("batch %d has same prev and post L1 message queue hashes but processed %d L1 messages in this batch", batchIndex, totalL1MessagesInBatch)
	}

	return nil
}

// sanityChecksBeforeConstructingTransaction performs sanity checks before constructing a transaction.
func (r *Layer2Relayer) sanityChecksBeforeConstructingTransaction(batchesToSubmit []*dbBatchWithChunks) error {
	if len(batchesToSubmit) == 0 {
		return fmt.Errorf("no batches to submit")
	}

	// Basic validation
	if err := r.validateBatchesBasic(batchesToSubmit); err != nil {
		return err
	}

	// Codec version validation
	if err := r.validateCodecVersions(batchesToSubmit); err != nil {
		return err
	}

	// Get previous chunk for continuity check
	prevChunk, err := r.getPreviousChunkForContinuity(batchesToSubmit[0])
	if err != nil {
		return err
	}

	// Validate each batch in detail
	if err := r.validateBatchesDetailed(batchesToSubmit, prevChunk); err != nil {
		return err
	}

	log.Info("Sanity check passed before constructing transaction", "batches count", len(batchesToSubmit))
	return nil
}

// validateBatchesBasic performs basic validation on all batches
func (r *Layer2Relayer) validateBatchesBasic(batchesToSubmit []*dbBatchWithChunks) error {
	for i, batch := range batchesToSubmit {
		if batch == nil || batch.Batch == nil {
			return fmt.Errorf("batch %d is nil", i)
		}

		if len(batch.Chunks) == 0 {
			return fmt.Errorf("batch %d has no chunks", batch.Batch.Index)
		}
	}
	return nil
}

// validateCodecVersions checks all batches have the same codec version
func (r *Layer2Relayer) validateCodecVersions(batchesToSubmit []*dbBatchWithChunks) error {
	firstBatchCodecVersion := batchesToSubmit[0].Batch.CodecVersion
	for _, batch := range batchesToSubmit {
		if batch.Batch.CodecVersion != firstBatchCodecVersion {
			return fmt.Errorf("batch %d has different codec version %d, expected %d", batch.Batch.Index, batch.Batch.CodecVersion, firstBatchCodecVersion)
		}
	}
	return nil
}

// getPreviousChunkForContinuity gets the previous chunk for block continuity check
func (r *Layer2Relayer) getPreviousChunkForContinuity(firstBatch *dbBatchWithChunks) (*orm.Chunk, error) {
	firstChunk := firstBatch.Chunks[0]
	if firstChunk.Index == 0 {
		return nil, fmt.Errorf("genesis chunk should not be in normal batch submission flow, chunk index: %d", firstChunk.Index)
	}

	prevChunk, err := r.chunkOrm.GetChunkByIndex(r.ctx, firstChunk.Index-1)
	if err != nil {
		return nil, fmt.Errorf("failed to get previous chunk %d for continuity check: %w", firstChunk.Index-1, err)
	}

	return prevChunk, nil
}

// validateBatchesDetailed performs detailed validation on each batch
func (r *Layer2Relayer) validateBatchesDetailed(batchesToSubmit []*dbBatchWithChunks, prevChunkFromPrevBatch *orm.Chunk) error {
	for i, batch := range batchesToSubmit {
		if err := r.validateSingleBatch(batch, i, batchesToSubmit, prevChunkFromPrevBatch); err != nil {
			return err
		}
	}
	return nil
}

// validateSingleBatch validates a single batch and its chunks
func (r *Layer2Relayer) validateSingleBatch(batch *dbBatchWithChunks, i int, allBatches []*dbBatchWithChunks, prevChunkFromPrevBatch *orm.Chunk) error {
	// Validate batch fields
	if err := r.validateBatchFields(batch, i, allBatches); err != nil {
		return err
	}

	// Validate message queue consistency
	if err := r.validateMessageQueueConsistency(batch.Batch.Index, batch.Chunks, common.HexToHash(batch.Batch.PrevL1MessageQueueHash), common.HexToHash(batch.Batch.PostL1MessageQueueHash)); err != nil {
		return err
	}

	// Validate chunks
	if err := r.validateBatchChunks(batch, i, allBatches, prevChunkFromPrevBatch); err != nil {
		return err
	}

	return nil
}

// validateBatchFields validates essential batch fields
func (r *Layer2Relayer) validateBatchFields(batch *dbBatchWithChunks, i int, allBatches []*dbBatchWithChunks) error {
	// Check essential batch fields are not zero values
	batchHash := common.HexToHash(batch.Batch.Hash)
	if batchHash == (common.Hash{}) {
		return fmt.Errorf("batch %d hash is zero", batch.Batch.Index)
	}

	if batch.Batch.Index == 0 {
		return fmt.Errorf("batch %d has zero index (only genesis batch should have index 0)", i)
	}

	// Check batch index is sequential
	if i > 0 {
		prevBatch := allBatches[i-1]
		if batch.Batch.Index != prevBatch.Batch.Index+1 {
			return fmt.Errorf("batch index is not sequential: prev batch index %d, current batch index %d", prevBatch.Batch.Index, batch.Batch.Index)
		}
	} else {
		// For the first batch, check continuity with the parent batch from database
		parentBatch, err := r.batchOrm.GetBatchByHash(r.ctx, batch.Batch.ParentBatchHash)
		if err != nil {
			return fmt.Errorf("failed to get parent batch %s for batch %d: %w", batch.Batch.ParentBatchHash, batch.Batch.Index, err)
		}
		if batch.Batch.Index != parentBatch.Index+1 {
			return fmt.Errorf("first batch index is not sequential with parent: parent batch index %d, current batch index %d", parentBatch.Index, batch.Batch.Index)
		}
	}

	parentBatchHash := common.HexToHash(batch.Batch.ParentBatchHash)
	if parentBatchHash == (common.Hash{}) {
		return fmt.Errorf("batch %d parent batch hash is zero", batch.Batch.Index)
	}

	stateRoot := common.HexToHash(batch.Batch.StateRoot)
	if stateRoot == (common.Hash{}) {
		return fmt.Errorf("batch %d state root is zero", batch.Batch.Index)
	}

	return nil
}

// validateBatchChunks validates all chunks in a batch
func (r *Layer2Relayer) validateBatchChunks(batch *dbBatchWithChunks, i int, allBatches []*dbBatchWithChunks, prevChunkFromPrevBatch *orm.Chunk) error {
	// Check all chunks in this batch have the same codec version as the batch
	for _, chunk := range batch.Chunks {
		if chunk.CodecVersion != batch.Batch.CodecVersion {
			return fmt.Errorf("batch %d chunk %d has different codec version %d, expected %d", batch.Batch.Index, chunk.Index, chunk.CodecVersion, batch.Batch.CodecVersion)
		}
	}

	for j, chunk := range batch.Chunks {
		if err := r.validateSingleChunk(chunk, j, batch, i, allBatches, prevChunkFromPrevBatch); err != nil {
			return err
		}
	}

	return nil
}

// validateSingleChunk validates a single chunk
func (r *Layer2Relayer) validateSingleChunk(chunk *orm.Chunk, chunkIndex int, batch *dbBatchWithChunks, i int, allBatches []*dbBatchWithChunks, prevChunkFromPrevBatch *orm.Chunk) error {
	if chunk == nil {
		return fmt.Errorf("batch %d chunk %d is nil", batch.Batch.Index, chunkIndex)
	}

	chunkHash := common.HexToHash(chunk.Hash)
	if chunkHash == (common.Hash{}) {
		return fmt.Errorf("batch %d chunk %d hash is zero", batch.Batch.Index, chunk.Index)
	}

	// Get previous chunk for continuity check
	var prevChunk *orm.Chunk
	if chunkIndex > 0 {
		prevChunk = batch.Chunks[chunkIndex-1]
	} else if i == 0 {
		prevChunk = prevChunkFromPrevBatch
	} else if i > 0 {
		// Use the last chunk from the previous batch
		prevBatch := allBatches[i-1]
		prevChunk = prevBatch.Chunks[len(prevBatch.Chunks)-1]
	}

	// Check chunk index is sequential
	if chunk.Index != prevChunk.Index+1 {
		return fmt.Errorf("batch %d chunk %d index is not sequential: prev chunk index %d, current chunk index %d", batch.Batch.Index, chunkIndex, prevChunk.Index, chunk.Index)
	}

	// Check L1 messages popped continuity
	expectedPoppedBefore := prevChunk.TotalL1MessagesPoppedBefore + prevChunk.TotalL1MessagesPoppedInChunk
	if chunk.TotalL1MessagesPoppedBefore != expectedPoppedBefore {
		return fmt.Errorf("batch %d chunk %d L1 messages popped before is incorrect: expected %d, got %d",
			batch.Batch.Index, chunk.Index, expectedPoppedBefore, chunk.TotalL1MessagesPoppedBefore)
	}

	if chunk.StartBlockNumber == 0 && chunk.EndBlockNumber == 0 {
		return fmt.Errorf("batch %d chunk %d has zero block range", batch.Batch.Index, chunk.Index)
	}

	if chunk.StartBlockNumber > chunk.EndBlockNumber {
		return fmt.Errorf("batch %d chunk %d has invalid block range: start %d > end %d", batch.Batch.Index, chunk.Index, chunk.StartBlockNumber, chunk.EndBlockNumber)
	}

	// Check chunk hash fields
	startBlockHash := common.HexToHash(chunk.StartBlockHash)
	if startBlockHash == (common.Hash{}) {
		return fmt.Errorf("batch %d chunk %d start block hash is zero", batch.Batch.Index, chunk.Index)
	}

	endBlockHash := common.HexToHash(chunk.EndBlockHash)
	if endBlockHash == (common.Hash{}) {
		return fmt.Errorf("batch %d chunk %d end block hash is zero", batch.Batch.Index, chunk.Index)
	}

	// Check chunk continuity: previous chunk's end block number + 1 should equal current chunk's start block number
	if prevChunk.EndBlockNumber+1 != chunk.StartBlockNumber {
		return fmt.Errorf("batch %d chunk %d is not continuous with previous chunk: prev chunk %d end block %d, current chunk start block %d", batch.Batch.Index, chunk.Index, prevChunk.Index, prevChunk.EndBlockNumber, chunk.StartBlockNumber)
	}

	return nil
}
