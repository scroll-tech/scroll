package relayer

import (
	"fmt"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
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

	// Get previous chunk for continuity check
	prevChunk, err := r.getPreviousChunkForContinuity(batchesToSubmit[0])
	if err != nil {
		return err
	}

	// Validate batches (including basic, codec versions, and detailed checks)
	if err := r.validateBatches(batchesToSubmit, prevChunk); err != nil {
		return err
	}

	log.Info("Sanity check passed before constructing transaction", "batches count", len(batchesToSubmit))
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

// validateBatches performs validation on all batches including basic checks, codec version consistency, and detailed checks.
func (r *Layer2Relayer) validateBatches(batchesToSubmit []*dbBatchWithChunks, initialPrevChunk *orm.Chunk) error {
	// Basic validation: ensure each batch and its chunks are non-empty.
	for i, batch := range batchesToSubmit {
		if batch == nil || batch.Batch == nil {
			return fmt.Errorf("batch %d is nil", i)
		}
		if len(batch.Chunks) == 0 {
			return fmt.Errorf("batch %d has no chunks", batch.Batch.Index)
		}
	}

	// Check that all batches have the same codec version.
	firstBatchCodecVersion := batchesToSubmit[0].Batch.CodecVersion
	for _, batch := range batchesToSubmit {
		if batch.Batch.CodecVersion != firstBatchCodecVersion {
			return fmt.Errorf("batch %d has different codec version %d, expected %d", batch.Batch.Index, batch.Batch.CodecVersion, firstBatchCodecVersion)
		}
	}

	// Validate each batch in detail, updating the previous chunk as we go.
	currentPrevChunk := initialPrevChunk
	for i, batch := range batchesToSubmit {
		if err := r.validateSingleBatch(batch, i, batchesToSubmit, currentPrevChunk); err != nil {
			return err
		}
		// Update the previous chunk to the last chunk of this batch for the next batch.
		currentPrevChunk = batch.Chunks[len(batch.Chunks)-1]
	}
	return nil
}

// validateSingleBatch validates a single batch and its chunks
func (r *Layer2Relayer) validateSingleBatch(batch *dbBatchWithChunks, i int, allBatches []*dbBatchWithChunks, prevChunk *orm.Chunk) error {
	// Validate batch fields
	if err := r.validateBatchFields(batch, i, allBatches); err != nil {
		return err
	}

	// Validate message queue consistency
	if err := r.validateMessageQueueConsistency(batch.Batch.Index, batch.Chunks, common.HexToHash(batch.Batch.PrevL1MessageQueueHash), common.HexToHash(batch.Batch.PostL1MessageQueueHash)); err != nil {
		return err
	}

	// Validate chunks
	if err := r.validateBatchChunks(batch, prevChunk); err != nil {
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
func (r *Layer2Relayer) validateBatchChunks(batch *dbBatchWithChunks, prevChunk *orm.Chunk) error {
	// Check codec version consistency.
	for _, chunk := range batch.Chunks {
		if chunk.CodecVersion != batch.Batch.CodecVersion {
			return fmt.Errorf("batch %d chunk %d has different codec version %d, expected %d", batch.Batch.Index, chunk.Index, chunk.CodecVersion, batch.Batch.CodecVersion)
		}
	}

	for j, chunk := range batch.Chunks {
		if err := r.validateSingleChunk(chunk, prevChunk); err != nil {
			return fmt.Errorf("batch %d chunk %d: %w", batch.Batch.Index, j, err)
		}
		// Update the previous chunk to the current one for the next chunk.
		prevChunk = chunk
	}

	return nil
}

// validateSingleChunk validates a single chunk
func (r *Layer2Relayer) validateSingleChunk(chunk *orm.Chunk, prevChunk *orm.Chunk) error {
	if chunk == nil {
		return fmt.Errorf("chunk is nil")
	}

	chunkHash := common.HexToHash(chunk.Hash)
	if chunkHash == (common.Hash{}) {
		return fmt.Errorf("chunk %d hash is zero", chunk.Index)
	}

	// Check chunk index is sequential
	if chunk.Index != prevChunk.Index+1 {
		return fmt.Errorf("chunk index is not sequential: prev chunk index %d, current chunk index %d", prevChunk.Index, chunk.Index)
	}

	// Check L1 messages popped continuity
	expectedPoppedBefore := prevChunk.TotalL1MessagesPoppedBefore + prevChunk.TotalL1MessagesPoppedInChunk
	if chunk.TotalL1MessagesPoppedBefore != expectedPoppedBefore {
		return fmt.Errorf("L1 messages popped before is incorrect: expected %d, got %d", expectedPoppedBefore, chunk.TotalL1MessagesPoppedBefore)
	}

	if chunk.StartBlockNumber == 0 && chunk.EndBlockNumber == 0 {
		return fmt.Errorf("chunk %d has zero block range", chunk.Index)
	}

	if chunk.StartBlockNumber > chunk.EndBlockNumber {
		return fmt.Errorf("chunk %d has invalid block range: start %d > end %d", chunk.Index, chunk.StartBlockNumber, chunk.EndBlockNumber)
	}

	// Check chunk hash fields
	startBlockHash := common.HexToHash(chunk.StartBlockHash)
	if startBlockHash == (common.Hash{}) {
		return fmt.Errorf("chunk %d start block hash is zero", chunk.Index)
	}

	endBlockHash := common.HexToHash(chunk.EndBlockHash)
	if endBlockHash == (common.Hash{}) {
		return fmt.Errorf("chunk %d end block hash is zero", chunk.Index)
	}

	// Check chunk continuity: previous chunk's end block number + 1 should equal current chunk's start block number
	if prevChunk.EndBlockNumber+1 != chunk.StartBlockNumber {
		return fmt.Errorf("chunk is not continuous with previous chunk %d: prev end block %d, current start block %d", prevChunk.Index, prevChunk.EndBlockNumber, chunk.StartBlockNumber)
	}

	return nil
}

func (r *Layer2Relayer) sanityChecksCommitBatchCodecV7CalldataAndBlobs(calldata []byte, blobs []*kzg4844.Blob, batchesToSubmit []*dbBatchWithChunks, firstBatch, lastBatch *orm.Batch,
) error {
	// Check blob count matches batch count
	if len(blobs) != len(batchesToSubmit) {
		return fmt.Errorf("blob count mismatch: got %d, want %d", len(blobs), len(batchesToSubmit))
	}

	// Parse calldata (after first 4 bytes: method selector)
	method := r.l1RollupABI.Methods["commitBatches"]
	if len(calldata) < 4 {
		return fmt.Errorf("calldata too short to contain method selector")
	}
	decoded, err := method.Inputs.Unpack(calldata[4:])
	if err != nil {
		return fmt.Errorf("failed to unpack commitBatches calldata: %w", err)
	}

	if len(decoded) != 3 {
		return fmt.Errorf("unexpected number of decoded parameters: got %d, want 3", len(decoded))
	}

	version, ok := decoded[0].(uint8)
	if !ok {
		return fmt.Errorf("failed to type assert version to uint8")
	}
	parentBatchHashB, ok := decoded[1].([32]uint8)
	if !ok {
		return fmt.Errorf("failed to type assert parentBatchHash to [32]uint8")
	}
	parentBatchHash := common.BytesToHash(parentBatchHashB[:])
	lastBatchHashB, ok := decoded[2].([32]uint8)
	if !ok {
		return fmt.Errorf("failed to type assert lastBatchHash to [32]uint8")
	}
	lastBatchHash := common.BytesToHash(lastBatchHashB[:])

	// Check version and batch hashes
	if version != uint8(firstBatch.CodecVersion) {
		return fmt.Errorf("sanity check failed: version mismatch: calldata=%d, db=%d", version, firstBatch.CodecVersion)
	}
	if parentBatchHash != common.HexToHash(firstBatch.ParentBatchHash) {
		return fmt.Errorf("sanity check failed: parentBatchHash mismatch: calldata=%s, db=%s", parentBatchHash.Hex(), firstBatch.ParentBatchHash)
	}
	if lastBatchHash != common.HexToHash(lastBatch.Hash) {
		return fmt.Errorf("sanity check failed: lastBatchHash mismatch: calldata=%s, db=%s", lastBatchHash.Hex(), lastBatch.Hash)
	}

	// Get codec for blob decoding
	codec, err := encoding.CodecFromVersion(encoding.CodecVersion(firstBatch.CodecVersion))
	if err != nil {
		return fmt.Errorf("failed to get codec: %w", err)
	}

	// Loop through each batch and blob, decode and compare
	for i, blob := range blobs {
		dbBatch := batchesToSubmit[i].Batch
		dbChunks := batchesToSubmit[i].Chunks

		// Collect all blocks for the batch
		var batchBlocks []*encoding.Block
		for _, c := range dbChunks {
			blocks, err := r.l2BlockOrm.GetL2BlocksInRange(r.ctx, c.StartBlockNumber, c.EndBlockNumber)
			if err != nil {
				return fmt.Errorf("failed to get blocks for batch %d chunk %d: %w", dbBatch.Index, c.Index, err)
			}
			batchBlocks = append(batchBlocks, blocks...)
		}

		// Decode blob payload
		payload, err := codec.DecodeBlob(blob)
		if err != nil {
			return fmt.Errorf("failed to decode blob for batch %d: %w", dbBatch.Index, err)
		}

		// Check L1 message queue hashes
		if payload.PrevL1MessageQueueHash() != common.HexToHash(dbBatch.PrevL1MessageQueueHash) {
			return fmt.Errorf("sanity check failed: prevL1MessageQueueHash mismatch for batch %d: decoded=%s, db=%s",
				dbBatch.Index, payload.PrevL1MessageQueueHash().Hex(), dbBatch.PrevL1MessageQueueHash)
		}
		if payload.PostL1MessageQueueHash() != common.HexToHash(dbBatch.PostL1MessageQueueHash) {
			return fmt.Errorf("sanity check failed: postL1MessageQueueHash mismatch for batch %d: decoded=%s, db=%s",
				dbBatch.Index, payload.PostL1MessageQueueHash().Hex(), dbBatch.PostL1MessageQueueHash)
		}

		// Compare block count and block numbers
		decodedBlocks := payload.Blocks()
		if len(decodedBlocks) != len(batchBlocks) {
			return fmt.Errorf("sanity check failed: block count mismatch in batch %d: decoded=%d, db=%d", dbBatch.Index, len(decodedBlocks), len(batchBlocks))
		}
		for j, b := range batchBlocks {
			if decodedBlocks[j].Number() != b.Header.Number.Uint64() {
				return fmt.Errorf("sanity check failed: block number mismatch in batch %d block %d: decoded=%d, db=%d",
					dbBatch.Index, j, decodedBlocks[j].Number(), b.Header.Number.Uint64())
			}
		}
	}

	// All checks passed
	return nil
}
