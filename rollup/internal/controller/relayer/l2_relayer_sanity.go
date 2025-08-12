package relayer

import (
	"fmt"
	"math/big"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"

	"scroll-tech/rollup/internal/orm"
)

// sanityChecksCommitBatchCodecV7CalldataAndBlobs performs comprehensive validation of the constructed
// transaction data (calldata and blobs) by parsing them and comparing against database records.
// This ensures the constructed transaction data is correct and consistent with the database state.
func (r *Layer2Relayer) sanityChecksCommitBatchCodecV7CalldataAndBlobs(calldata []byte, blobs []*kzg4844.Blob) error {
	calldataInfo, err := parseCommitBatchesCalldata(r.l1RollupABI, calldata)
	if err != nil {
		return fmt.Errorf("failed to parse calldata: %w", err)
	}

	batchesToValidate, l1MessagesWithBlockNumbers, err := r.getBatchesFromCalldata(calldataInfo)
	if err != nil {
		return fmt.Errorf("failed to get batches from database: %w", err)
	}

	if err := r.validateCalldataAndBlobsAgainstDatabase(calldataInfo, blobs, batchesToValidate, l1MessagesWithBlockNumbers); err != nil {
		return fmt.Errorf("calldata and blobs validation failed: %w", err)
	}

	if err := r.validateDatabaseConsistency(batchesToValidate); err != nil {
		return fmt.Errorf("database consistency validation failed: %w", err)
	}

	return nil
}

// CalldataInfo holds parsed information from commitBatches calldata
type CalldataInfo struct {
	Version         uint8
	ParentBatchHash common.Hash
	LastBatchHash   common.Hash
}

// parseCommitBatchesCalldata parses the commitBatches calldata and extracts key information
func parseCommitBatchesCalldata(abi *abi.ABI, calldata []byte) (*CalldataInfo, error) {
	method := abi.Methods["commitBatches"]
	decoded, err := method.Inputs.Unpack(calldata[4:])
	if err != nil {
		return nil, fmt.Errorf("failed to unpack commitBatches calldata: %w", err)
	}

	if len(decoded) != 3 {
		return nil, fmt.Errorf("unexpected number of decoded parameters: got %d, want 3", len(decoded))
	}

	version, ok := decoded[0].(uint8)
	if !ok {
		return nil, fmt.Errorf("failed to type assert version to uint8")
	}

	parentBatchHashB, ok := decoded[1].([32]uint8)
	if !ok {
		return nil, fmt.Errorf("failed to type assert parentBatchHash to [32]uint8")
	}
	parentBatchHash := common.BytesToHash(parentBatchHashB[:])

	lastBatchHashB, ok := decoded[2].([32]uint8)
	if !ok {
		return nil, fmt.Errorf("failed to type assert lastBatchHash to [32]uint8")
	}
	lastBatchHash := common.BytesToHash(lastBatchHashB[:])

	return &CalldataInfo{
		Version:         version,
		ParentBatchHash: parentBatchHash,
		LastBatchHash:   lastBatchHash,
	}, nil
}

// getBatchesFromCalldata retrieves the relevant batches from database based on calldata information
func (r *Layer2Relayer) getBatchesFromCalldata(info *CalldataInfo) ([]*dbBatchWithChunks, map[uint64][]*types.TransactionData, error) {
	// Get the parent batch to determine the starting point
	parentBatch, err := r.batchOrm.GetBatchByHash(r.ctx, info.ParentBatchHash.Hex())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get parent batch by hash %s: %w", info.ParentBatchHash.Hex(), err)
	}

	// Get the last batch to determine the ending point
	lastBatch, err := r.batchOrm.GetBatchByHash(r.ctx, info.LastBatchHash.Hex())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get last batch by hash %s: %w", info.LastBatchHash.Hex(), err)
	}

	// Get all batches in the range (parent+1 to last)
	firstBatchIndex := parentBatch.Index + 1
	lastBatchIndex := lastBatch.Index

	// Check if the range is valid
	if firstBatchIndex > lastBatchIndex {
		return nil, nil, fmt.Errorf("no batches found in range: first index %d, last index %d", firstBatchIndex, lastBatchIndex)
	}

	var batchesToValidate []*dbBatchWithChunks
	l1MessagesWithBlockNumbers := make(map[uint64][]*types.TransactionData)
	for batchIndex := firstBatchIndex; batchIndex <= lastBatchIndex; batchIndex++ {
		dbBatch, err := r.batchOrm.GetBatchByIndex(r.ctx, batchIndex)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get batch by index %d: %w", batchIndex, err)
		}

		// Get chunks for this batch
		dbChunks, err := r.chunkOrm.GetChunksInRange(r.ctx, dbBatch.StartChunkIndex, dbBatch.EndChunkIndex)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get chunks for batch %d: %w", batchIndex, err)
		}

		batchesToValidate = append(batchesToValidate, &dbBatchWithChunks{
			Batch:  dbBatch,
			Chunks: dbChunks,
		})

		// If there are L1 messages in this batch, retrieve L1 messages with block numbers
		for _, chunk := range dbChunks {
			if chunk.TotalL1MessagesPoppedInChunk > 0 {
				blockWithL1Messages, err := r.l2BlockOrm.GetL2BlocksInRange(r.ctx, chunk.StartBlockNumber, chunk.EndBlockNumber)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to get L2 blocks for chunk %d: %w", chunk.Index, err)
				}
				for _, block := range blockWithL1Messages {
					bn := block.Header.Number.Uint64()
					seenL2 := false
					for _, tx := range block.Transactions {
						if tx.Type == types.L1MessageTxType {
							if seenL2 {
								// Invariant violated: found an L1 after an L2 in the same block.
								return nil, nil, fmt.Errorf("L1 message after L2 tx in block %d", bn)
							}
							l1MessagesWithBlockNumbers[bn] = append(l1MessagesWithBlockNumbers[bn], tx)
						} else {
							seenL2 = true
						}
					}
				}
			}
		}
	}
	return batchesToValidate, l1MessagesWithBlockNumbers, nil
}

// validateDatabaseConsistency performs comprehensive validation of database records
func (r *Layer2Relayer) validateDatabaseConsistency(batchesToValidate []*dbBatchWithChunks) error {
	if len(batchesToValidate) == 0 {
		return fmt.Errorf("no batches to validate")
	}

	// Get previous chunk for continuity check
	firstChunk := batchesToValidate[0].Chunks[0]
	if firstChunk.Index == 0 {
		return fmt.Errorf("genesis chunk should not be in normal batch submission flow, chunk index: %d", firstChunk.Index)
	}

	prevChunk, err := r.chunkOrm.GetChunkByIndex(r.ctx, firstChunk.Index-1)
	if err != nil {
		return fmt.Errorf("failed to get previous chunk %d for continuity check: %w", firstChunk.Index-1, err)
	}

	firstBatchCodecVersion := batchesToValidate[0].Batch.CodecVersion
	for i, batch := range batchesToValidate {
		// Validate codec version consistency
		if batch.Batch.CodecVersion != firstBatchCodecVersion {
			return fmt.Errorf("batch %d has different codec version %d, expected %d", batch.Batch.Index, batch.Batch.CodecVersion, firstBatchCodecVersion)
		}

		// Validate individual batch
		if err := r.validateSingleBatchConsistency(batch, i, batchesToValidate); err != nil {
			return err
		}

		// Validate chunks in this batch
		if err := r.validateBatchChunksConsistency(batch, prevChunk); err != nil {
			return err
		}

		// Update prevChunk to the last chunk of this batch for next iteration
		if len(batch.Chunks) == 0 {
			return fmt.Errorf("batch %d has no chunks", batch.Batch.Index)
		}
		prevChunk = batch.Chunks[len(batch.Chunks)-1]
	}

	return nil
}

// validateSingleBatchConsistency validates a single batch's consistency
func (r *Layer2Relayer) validateSingleBatchConsistency(batch *dbBatchWithChunks, i int, allBatches []*dbBatchWithChunks) error {
	if batch == nil || batch.Batch == nil {
		return fmt.Errorf("batch %d is nil", i)
	}

	if len(batch.Chunks) == 0 {
		return fmt.Errorf("batch %d has no chunks", batch.Batch.Index)
	}

	// Validate essential batch fields
	batchHash := common.HexToHash(batch.Batch.Hash)
	if batchHash == (common.Hash{}) {
		return fmt.Errorf("batch %d hash is zero", batch.Batch.Index)
	}

	if batch.Batch.Index == 0 {
		return fmt.Errorf("batch %d has zero index (only genesis batch should have index 0)", i)
	}

	parentBatchHash := common.HexToHash(batch.Batch.ParentBatchHash)
	if parentBatchHash == (common.Hash{}) {
		return fmt.Errorf("batch %d parent batch hash is zero", batch.Batch.Index)
	}

	stateRoot := common.HexToHash(batch.Batch.StateRoot)
	if stateRoot == (common.Hash{}) {
		return fmt.Errorf("batch %d state root is zero", batch.Batch.Index)
	}

	// Check batch index continuity
	if i > 0 {
		prevBatch := allBatches[i-1]
		if batch.Batch.Index != prevBatch.Batch.Index+1 {
			return fmt.Errorf("batch index is not sequential: prev batch index %d, current batch index %d", prevBatch.Batch.Index, batch.Batch.Index)
		}
		if parentBatchHash != common.HexToHash(prevBatch.Batch.Hash) {
			return fmt.Errorf("parent batch hash does not match previous batch hash: expected %s, got %s", prevBatch.Batch.Hash, batch.Batch.ParentBatchHash)
		}
	} else {
		// For the first batch, verify continuity with parent batch from database
		parentBatch, err := r.batchOrm.GetBatchByHash(r.ctx, batch.Batch.ParentBatchHash)
		if err != nil {
			return fmt.Errorf("failed to get parent batch %s for batch %d: %w", batch.Batch.ParentBatchHash, batch.Batch.Index, err)
		}
		if batch.Batch.Index != parentBatch.Index+1 {
			return fmt.Errorf("first batch index is not sequential with parent: parent batch index %d, current batch index %d", parentBatch.Index, batch.Batch.Index)
		}
	}

	// Validate L1 message queue consistency
	if err := r.validateMessageQueueConsistency(batch.Batch.Index, batch.Chunks, common.HexToHash(batch.Batch.PrevL1MessageQueueHash), common.HexToHash(batch.Batch.PostL1MessageQueueHash)); err != nil {
		return err
	}

	return nil
}

// validateBatchChunksConsistency validates chunks within a batch
func (r *Layer2Relayer) validateBatchChunksConsistency(batch *dbBatchWithChunks, prevChunk *orm.Chunk) error {
	// Check codec version consistency between chunks and batch
	for _, chunk := range batch.Chunks {
		if chunk.CodecVersion != batch.Batch.CodecVersion {
			return fmt.Errorf("batch %d chunk %d has different codec version %d, expected %d", batch.Batch.Index, chunk.Index, chunk.CodecVersion, batch.Batch.CodecVersion)
		}
	}

	// Validate each chunk individually
	currentPrevChunk := prevChunk
	for j, chunk := range batch.Chunks {
		if err := r.validateSingleChunkConsistency(chunk, currentPrevChunk); err != nil {
			return fmt.Errorf("batch %d chunk %d: %w", batch.Batch.Index, j, err)
		}
		currentPrevChunk = chunk
	}

	return nil
}

// validateSingleChunkConsistency validates a single chunk
func (r *Layer2Relayer) validateSingleChunkConsistency(chunk *orm.Chunk, prevChunk *orm.Chunk) error {
	if chunk == nil {
		return fmt.Errorf("chunk is nil")
	}

	chunkHash := common.HexToHash(chunk.Hash)
	if chunkHash == (common.Hash{}) {
		return fmt.Errorf("chunk %d hash is zero", chunk.Index)
	}

	// Check chunk index continuity
	if chunk.Index != prevChunk.Index+1 {
		return fmt.Errorf("chunk index is not sequential: prev chunk index %d, current chunk index %d", prevChunk.Index, chunk.Index)
	}

	// Validate block range
	if chunk.StartBlockNumber == 0 && chunk.EndBlockNumber == 0 {
		return fmt.Errorf("chunk %d has zero block range", chunk.Index)
	}

	if chunk.StartBlockNumber > chunk.EndBlockNumber {
		return fmt.Errorf("chunk %d has invalid block range: start %d > end %d", chunk.Index, chunk.StartBlockNumber, chunk.EndBlockNumber)
	}

	// Check hash fields
	startBlockHash := common.HexToHash(chunk.StartBlockHash)
	if startBlockHash == (common.Hash{}) {
		return fmt.Errorf("chunk %d start block hash is zero", chunk.Index)
	}

	endBlockHash := common.HexToHash(chunk.EndBlockHash)
	if endBlockHash == (common.Hash{}) {
		return fmt.Errorf("chunk %d end block hash is zero", chunk.Index)
	}

	// Check block continuity with previous chunk
	if prevChunk.EndBlockNumber+1 != chunk.StartBlockNumber {
		return fmt.Errorf("chunk is not continuous with previous chunk %d: prev end block %d, current start block %d", prevChunk.Index, prevChunk.EndBlockNumber, chunk.StartBlockNumber)
	}

	// Check L1 messages continuity
	expectedPoppedBefore := prevChunk.TotalL1MessagesPoppedBefore + prevChunk.TotalL1MessagesPoppedInChunk
	if chunk.TotalL1MessagesPoppedBefore != expectedPoppedBefore {
		return fmt.Errorf("L1 messages popped before is incorrect: expected %d, got %d", expectedPoppedBefore, chunk.TotalL1MessagesPoppedBefore)
	}

	return nil
}

// validateCalldataAndBlobsAgainstDatabase validates calldata and blobs against database records
func (r *Layer2Relayer) validateCalldataAndBlobsAgainstDatabase(calldataInfo *CalldataInfo, blobs []*kzg4844.Blob, batchesToValidate []*dbBatchWithChunks, l1MessagesWithBlockNumbers map[uint64][]*types.TransactionData) error {
	// Validate blobs
	if len(blobs) == 0 {
		return fmt.Errorf("no blobs provided")
	}

	// Validate blob count
	if len(blobs) != len(batchesToValidate) {
		return fmt.Errorf("blob count mismatch: got %d blobs, expected %d batches", len(blobs), len(batchesToValidate))
	}

	// Get first and last batches for validation, length check is already done above
	firstBatch := batchesToValidate[0].Batch
	lastBatch := batchesToValidate[len(batchesToValidate)-1].Batch

	// Validate codec version
	if calldataInfo.Version != uint8(firstBatch.CodecVersion) {
		return fmt.Errorf("version mismatch: calldata=%d, db=%d", calldataInfo.Version, firstBatch.CodecVersion)
	}

	// Validate parent batch hash
	if calldataInfo.ParentBatchHash != common.HexToHash(firstBatch.ParentBatchHash) {
		return fmt.Errorf("parentBatchHash mismatch: calldata=%s, db=%s", calldataInfo.ParentBatchHash.Hex(), firstBatch.ParentBatchHash)
	}

	// Validate last batch hash
	if calldataInfo.LastBatchHash != common.HexToHash(lastBatch.Hash) {
		return fmt.Errorf("lastBatchHash mismatch: calldata=%s, db=%s", calldataInfo.LastBatchHash.Hex(), lastBatch.Hash)
	}

	// Get codec for blob decoding
	codec, err := encoding.CodecFromVersion(encoding.CodecVersion(firstBatch.CodecVersion))
	if err != nil {
		return fmt.Errorf("failed to get codec: %w", err)
	}

	// Validate each blob against its corresponding batch
	for i, blob := range blobs {
		dbBatch := batchesToValidate[i].Batch
		if err := r.validateSingleBlobAgainstBatch(blob, dbBatch, codec, l1MessagesWithBlockNumbers); err != nil {
			return fmt.Errorf("blob validation failed for batch %d: %w", dbBatch.Index, err)
		}
	}

	return nil
}

// validateSingleBlobAgainstBatch validates a single blob against its batch data
func (r *Layer2Relayer) validateSingleBlobAgainstBatch(blob *kzg4844.Blob, dbBatch *orm.Batch, codec encoding.Codec, l1MessagesWithBlockNumbers map[uint64][]*types.TransactionData) error {
	// Decode blob payload
	payload, err := codec.DecodeBlob(blob)
	if err != nil {
		return fmt.Errorf("failed to decode blob: %w", err)
	}

	// Validate batch hash
	daBatch, err := assembleDABatchFromPayload(payload, dbBatch, codec, l1MessagesWithBlockNumbers)
	if err != nil {
		return fmt.Errorf("failed to assemble batch from payload: %w", err)
	}

	if daBatch.Hash() != common.HexToHash(dbBatch.Hash) {
		return fmt.Errorf("batch hash mismatch: decoded from blob=%s, db=%s", daBatch.Hash().Hex(), dbBatch.Hash)
	}

	return nil
}

// validateMessageQueueConsistency validates L1 message queue hash consistency
func (r *Layer2Relayer) validateMessageQueueConsistency(batchIndex uint64, chunks []*orm.Chunk, prevL1MsgQueueHash common.Hash, postL1MsgQueueHash common.Hash) error {
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

func assembleDABatchFromPayload(payload encoding.DABlobPayload, dbBatch *orm.Batch, codec encoding.Codec, l1MessagesWithBlockNumbers map[uint64][]*types.TransactionData) (encoding.DABatch, error) {
	blocks, err := assembleBlocksFromPayload(payload, l1MessagesWithBlockNumbers)
	if err != nil {
		return nil, fmt.Errorf("failed to assemble blocks from payload batch_index=%d codec_version=%d parent_batch_hash=%s: %w", dbBatch.Index, dbBatch.CodecVersion, dbBatch.ParentBatchHash, err)
	}
	batch := &encoding.Batch{
		Index:                  dbBatch.Index,                             // The database provides only batch index, other fields are derived from blob payload
		ParentBatchHash:        common.HexToHash(dbBatch.ParentBatchHash), // The first batch's parent hash is verified with calldata, subsequent batches are linked via dbBatch.ParentBatchHash and verified in database consistency checks
		PrevL1MessageQueueHash: payload.PrevL1MessageQueueHash(),
		PostL1MessageQueueHash: payload.PostL1MessageQueueHash(),
		Blocks:                 blocks,
		Chunks: []*encoding.Chunk{ // One chunk for this batch to pass sanity checks when building DABatch
			{
				Blocks:                 blocks,
				PrevL1MessageQueueHash: payload.PrevL1MessageQueueHash(),
				PostL1MessageQueueHash: payload.PostL1MessageQueueHash(),
			},
		},
	}
	daBatch, err := codec.NewDABatch(batch)
	if err != nil {
		return nil, fmt.Errorf("failed to build DABatch batch_index=%d codec_version=%d parent_batch_hash=%s: %w", dbBatch.Index, dbBatch.CodecVersion, dbBatch.ParentBatchHash, err)
	}
	return daBatch, nil
}

func assembleBlocksFromPayload(payload encoding.DABlobPayload, l1MessagesWithBlockNumbers map[uint64][]*types.TransactionData) ([]*encoding.Block, error) {
	daBlocks := payload.Blocks()
	txns := payload.Transactions()
	if len(daBlocks) != len(txns) {
		return nil, fmt.Errorf("mismatched number of blocks and transactions: %d blocks, %d transactions", len(daBlocks), len(txns))
	}
	blocks := make([]*encoding.Block, len(daBlocks))
	for i := range daBlocks {
		blocks[i] = &encoding.Block{
			Header: &types.Header{
				Number:   new(big.Int).SetUint64(daBlocks[i].Number()),
				Time:     daBlocks[i].Timestamp(),
				BaseFee:  daBlocks[i].BaseFee(),
				GasLimit: daBlocks[i].GasLimit(),
			},
		}
		if l1Messages, ok := l1MessagesWithBlockNumbers[daBlocks[i].Number()]; ok {
			blocks[i].Transactions = l1Messages
		}
		blocks[i].Transactions = append(blocks[i].Transactions, encoding.TxsToTxsData(txns[i])...)
	}
	return blocks, nil
}
