package main

import (
	"context"
	"math/rand"
	"sort"

	"gorm.io/gorm"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/database"
	"scroll-tech/rollup/internal/orm"
	"scroll-tech/rollup/internal/utils"
)

type importRecord struct {
	Chunk  []string `json:"chunks"`
	Batch  []string `json:"batches"`
	Bundle []string `json:"bundles"`
}

func randomPickKfromN(n, k int, rng *rand.Rand) []int {
	ret := make([]int, n-1)
	for i := 1; i < n; i++ {
		ret[i-1] = i
	}

	rng.Shuffle(len(ret), func(i, j int) {
		ret[i], ret[j] = ret[j], ret[i]
	})

	ret = ret[:k-1]
	sort.Ints(ret)

	return ret
}

func importData(ctx context.Context, beginBlk, endBlk uint64, chkNum, batchNum, bundleNum int, seed int64) (*importRecord, error) {

	db, err := database.InitDB(cfg.DBConfig)
	if err != nil {
		return nil, err
	}
	ret := &importRecord{}
	// Create a new random source with the provided seed
	source := rand.NewSource(seed)
	rng := rand.New(source)

	chkSepIdx := randomPickKfromN(int(endBlk-beginBlk)+1, chkNum, rng)
	chkSep := make([]uint64, len(chkSepIdx))
	for i, ind := range chkSepIdx {
		chkSep[i] = beginBlk + uint64(ind)
	}
	chkSep = append(chkSep, endBlk)

	log.Info("separated chunk", "border", chkSep)
	head := beginBlk
	lastMsgHash := common.Hash{}

	ormChks := make([]*orm.Chunk, 0, chkNum)
	encChks := make([]*encoding.Chunk, 0, chkNum)
	for _, edBlk := range chkSep {
		ormChk, chk, err := importChunk(ctx, db, head, edBlk-1, lastMsgHash)
		if err != nil {
			return nil, err
		}
		lastMsgHash = chk.PostL1MessageQueueHash
		ormChks = append(ormChks, ormChk)
		encChks = append(encChks, chk)
		head = edBlk
	}

	for _, chk := range ormChks {
		ret.Chunk = append(ret.Chunk, chk.Hash)
	}

	batchSep := randomPickKfromN(chkNum, batchNum, rng)
	batchSep = append(batchSep, chkNum)
	log.Info("separated batch", "border", batchSep)

	headChk := int(0)
	batches := make([]*orm.Batch, 0, batchNum)
	var lastBatch *orm.Batch
	for _, endChk := range batchSep {
		batch, err := importBatch(ctx, db, ormChks[headChk:endChk], encChks[headChk:endChk], lastBatch)
		if err != nil {
			return nil, err
		}
		lastBatch = batch
		batches = append(batches, batch)
		headChk = endChk
	}

	for _, batch := range batches {
		ret.Batch = append(ret.Batch, batch.Hash)
	}

	bundleSep := randomPickKfromN(batchNum, bundleNum, rng)
	bundleSep = append(bundleSep, batchNum)
	log.Info("separated bundle", "border", bundleSep)

	headBatch := int(0)
	for _, endBatch := range bundleSep {
		hash, err := importBundle(ctx, db, batches[headBatch:endBatch])
		if err != nil {
			return nil, err
		}
		ret.Bundle = append(ret.Bundle, hash)
		headBatch = endBatch
	}

	return ret, nil
}

func importChunk(ctx context.Context, db *gorm.DB, beginBlk, endBlk uint64, prevMsgQueueHash common.Hash) (*orm.Chunk, *encoding.Chunk, error) {
	nblk := int(endBlk-beginBlk) + 1
	blockOrm := orm.NewL2Block(db)

	blks, err := blockOrm.GetL2BlocksGEHeight(ctx, beginBlk, nblk)

	if err != nil {
		return nil, nil, err
	}

	postHash, err := encoding.MessageQueueV2ApplyL1MessagesFromBlocks(prevMsgQueueHash, blks)
	if err != nil {
		return nil, nil, err
	}

	theChunk := &encoding.Chunk{
		Blocks:                 blks,
		PrevL1MessageQueueHash: prevMsgQueueHash,
		PostL1MessageQueueHash: postHash,
	}
	chunkOrm := orm.NewChunk(db)

	dbChk, err := chunkOrm.InsertChunk(ctx, theChunk, codecCfg, utils.ChunkMetrics{})
	if err != nil {
		return nil, nil, err
	}
	err = blockOrm.UpdateChunkHashInRange(ctx, beginBlk, endBlk, dbChk.Hash)
	if err != nil {
		return nil, nil, err
	}
	log.Info("insert chunk", "From", beginBlk, "To", endBlk)
	return dbChk, theChunk, nil
}

func importBatch(ctx context.Context, db *gorm.DB, chks []*orm.Chunk, encChks []*encoding.Chunk, last *orm.Batch) (*orm.Batch, error) {

	batchOrm := orm.NewBatch(db)

	var index uint64
	var parentHash common.Hash
	if last != nil {
		index = last.Index + 1
		parentHash = common.HexToHash(last.Hash)
	}

	var blks []*encoding.Block
	for _, chk := range encChks {
		blks = append(blks, chk.Blocks...)
	}

	batch := &encoding.Batch{
		Index:                      index,
		TotalL1MessagePoppedBefore: chks[0].TotalL1MessagesPoppedBefore,
		ParentBatchHash:            parentHash,
		Chunks:                     encChks,
		Blocks:                     blks,
	}

	dbBatch, err := batchOrm.InsertBatch(ctx, batch, codecCfg, utils.BatchMetrics{})
	if err != nil {
		return nil, err
	}
	err = orm.NewChunk(db).UpdateBatchHashInRange(ctx, chks[0].Index, chks[len(chks)-1].Index, dbBatch.Hash)
	if err != nil {
		return nil, err
	}

	log.Info("insert batch", "index", index)
	return dbBatch, nil
}

func importBundle(ctx context.Context, db *gorm.DB, batches []*orm.Batch) (string, error) {

	bundleOrm := orm.NewBundle(db)
	bundle, err := bundleOrm.InsertBundle(ctx, batches, codecCfg)
	if err != nil {
		return "", err
	}
	err = orm.NewBatch(db).UpdateBundleHashInRange(ctx, batches[0].Index, batches[len(batches)-1].Index, bundle.Hash)
	if err != nil {
		return "", err
	}

	log.Info("insert bundle", "hash", bundle.Hash)
	return bundle.Hash, nil
}
