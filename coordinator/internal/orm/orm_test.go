package orm

import (
	"context"
	"math/big"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"scroll-tech/database/migrate"

	"scroll-tech/common/testcontainers"
	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
	"scroll-tech/common/utils"
)

var (
	testApps      *testcontainers.TestcontainerApps
	db            *gorm.DB
	proverTaskOrm *ProverTask
)

func TestMain(m *testing.M) {
	t := &testing.T{}
	defer func() {
		if testApps != nil {
			testApps.Free()
			tearDownEnv(t)
		}
	}()
	m.Run()
}

func setupEnv(t *testing.T) {
	testApps = testcontainers.NewTestcontainerApps()
	assert.NoError(t, testApps.StartPostgresContainer())

	var err error
	db, err = testApps.GetGormDBClient()
	assert.NoError(t, err)
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	proverTaskOrm = NewProverTask(db)
}

func tearDownEnv(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	sqlDB.Close()
}

func TestProverTaskOrm(t *testing.T) {
	setupEnv(t)

	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	reward := big.NewInt(0)
	reward.SetString("18446744073709551616", 10) // 1 << 64, uint64 maximum 1<<64 -1

	proverTask := ProverTask{
		TaskType:        int16(message.ProofTypeChunk),
		TaskID:          "test-hash",
		ProverName:      "prover-0",
		ProverPublicKey: "0",
		ProvingStatus:   int16(types.ProverAssigned),
		Reward:          decimal.NewFromBigInt(reward, 0),
		AssignedAt:      utils.NowUTC(),
	}

	err = proverTaskOrm.InsertProverTask(context.Background(), &proverTask)
	assert.NoError(t, err)
	proverTasks, err := proverTaskOrm.GetProverTasksByHashes(context.Background(), message.ProofTypeChunk, []string{"test-hash"})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(proverTasks))
	assert.Equal(t, proverTask.ProverName, proverTasks[0].ProverName)
	assert.NotEqual(t, proverTask.UUID.String(), "00000000-0000-0000-0000-000000000000")

	// test decimal reward, get reward
	resultReward := proverTasks[0].Reward.BigInt()
	assert.Equal(t, resultReward, reward)
	assert.Equal(t, resultReward.String(), "18446744073709551616")

	proverTask.ProvingStatus = int16(types.ProverProofValid)
	proverTask.AssignedAt = utils.NowUTC()
	err = proverTaskOrm.InsertProverTask(context.Background(), &proverTask)
	assert.Error(t, err)
}

func TestProverTaskOrmUint256(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	// test reward for uint256 maximum 1 << 256 -1 :115792089237316195423570985008687907853269984665640564039457584007913129639935
	rewardUint256 := big.NewInt(0)
	rewardUint256.SetString("115792089237316195423570985008687907853269984665640564039457584007913129639935", 10)
	proverTask := ProverTask{
		TaskType:        int16(message.ProofTypeChunk),
		TaskID:          "test-hash",
		ProverName:      "prover-0",
		ProverPublicKey: "0",
		ProvingStatus:   int16(types.ProverAssigned),
		Reward:          decimal.NewFromBigInt(rewardUint256, 0),
		AssignedAt:      utils.NowUTC(),
	}

	err = proverTaskOrm.InsertProverTask(context.Background(), &proverTask)
	assert.NoError(t, err)
	assert.NotEqual(t, proverTask.UUID.String(), "00000000-0000-0000-0000-000000000000")
	proverTasksUint256, err := proverTaskOrm.GetProverTasksByHashes(context.Background(), message.ProofTypeChunk, []string{"test-hash"})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(proverTasksUint256))
	resultRewardUint256 := proverTasksUint256[0].Reward.BigInt()
	assert.Equal(t, resultRewardUint256, rewardUint256)
	assert.Equal(t, resultRewardUint256.String(), "115792089237316195423570985008687907853269984665640564039457584007913129639935")
}

func TestChunkRefundAttemptsByHash(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	chunkOrm := NewChunk(db)
	insertChunk := func(index uint64, hash string, provingStatus, totalAttempts, activeAttempts int16) {
		execErr := db.Exec(`INSERT INTO chunk (index, hash, start_block_number, start_block_hash, end_block_number, end_block_hash,
			total_l1_messages_popped_before, total_l1_messages_popped_in_chunk, start_block_time, parent_chunk_hash, state_root,
			parent_chunk_state_root, withdraw_root, total_l2_tx_gas, total_l2_tx_num, total_l1_commit_calldata_size, total_l1_commit_gas,
			proving_status, total_attempts, active_attempts)
			VALUES (?, ?, 1, '0x1', 2, '0x2', 0, 0, 0, '0x0', '0x3', '0x0', '0x4', 0, 0, 0, 0, ?, ?, ?)`,
			index, hash, provingStatus, totalAttempts, activeAttempts).Error
		assert.NoError(t, execErr)
	}

	// (a) charge via UpdateChunkAttempts then refund -> both counters back to 0, status back to unassigned
	insertChunk(1, "0xchunk-a", int16(types.ProvingTaskUnassigned), 0, 0)
	rowsAffected, err := chunkOrm.UpdateChunkAttempts(context.Background(), 1, 0, 0)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)
	active, total, err := chunkOrm.GetAttemptsByHash(context.Background(), "0xchunk-a")
	assert.NoError(t, err)
	assert.Equal(t, int16(1), active)
	assert.Equal(t, int16(1), total)
	status, err := chunkOrm.GetProvingStatusByHash(context.Background(), "0xchunk-a")
	assert.NoError(t, err)
	assert.Equal(t, types.ProvingTaskAssigned, status)

	assert.NoError(t, chunkOrm.RefundAttemptsByHash(context.Background(), "0xchunk-a"))
	active, total, err = chunkOrm.GetAttemptsByHash(context.Background(), "0xchunk-a")
	assert.NoError(t, err)
	assert.Equal(t, int16(0), active)
	assert.Equal(t, int16(0), total)
	status, err = chunkOrm.GetProvingStatusByHash(context.Background(), "0xchunk-a")
	assert.NoError(t, err)
	assert.Equal(t, types.ProvingTaskUnassigned, status)

	// (b) refund on a verified row -> no-op
	insertChunk(2, "0xchunk-b", int16(types.ProvingTaskVerified), 1, 1)
	assert.NoError(t, chunkOrm.RefundAttemptsByHash(context.Background(), "0xchunk-b"))
	active, total, err = chunkOrm.GetAttemptsByHash(context.Background(), "0xchunk-b")
	assert.NoError(t, err)
	assert.Equal(t, int16(1), active)
	assert.Equal(t, int16(1), total)
	status, err = chunkOrm.GetProvingStatusByHash(context.Background(), "0xchunk-b")
	assert.NoError(t, err)
	assert.Equal(t, types.ProvingTaskVerified, status)

	// (c) refund when attempts are already 0 -> no underflow, row unchanged
	insertChunk(3, "0xchunk-c", int16(types.ProvingTaskAssigned), 0, 0)
	assert.NoError(t, chunkOrm.RefundAttemptsByHash(context.Background(), "0xchunk-c"))
	active, total, err = chunkOrm.GetAttemptsByHash(context.Background(), "0xchunk-c")
	assert.NoError(t, err)
	assert.Equal(t, int16(0), active)
	assert.Equal(t, int16(0), total)
	status, err = chunkOrm.GetProvingStatusByHash(context.Background(), "0xchunk-c")
	assert.NoError(t, err)
	assert.Equal(t, types.ProvingTaskAssigned, status)
}

func TestBatchRefundAttemptsByHash(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	batchOrm := NewBatch(db)
	insertBatch := func(index uint64, hash string, provingStatus, totalAttempts, activeAttempts int16) {
		execErr := db.Exec(`INSERT INTO batch (index, hash, start_chunk_index, start_chunk_hash, end_chunk_index, end_chunk_hash,
			state_root, withdraw_root, parent_batch_hash, batch_header, proving_status, total_attempts, active_attempts)
			VALUES (?, ?, 1, '0x1', 2, '0x2', '0x3', '0x4', '0x0', ?, ?, ?, ?)`,
			index, hash, []byte{0}, provingStatus, totalAttempts, activeAttempts).Error
		assert.NoError(t, execErr)
	}

	// (a) charge via UpdateBatchAttempts then refund -> both counters back to 0, status back to unassigned
	insertBatch(1, "0xbatch-a", int16(types.ProvingTaskUnassigned), 0, 0)
	rowsAffected, err := batchOrm.UpdateBatchAttempts(context.Background(), 1, 0, 0)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)
	active, total, err := batchOrm.GetAttemptsByHash(context.Background(), "0xbatch-a")
	assert.NoError(t, err)
	assert.Equal(t, int16(1), active)
	assert.Equal(t, int16(1), total)
	status, err := batchOrm.GetProvingStatusByHash(context.Background(), "0xbatch-a")
	assert.NoError(t, err)
	assert.Equal(t, types.ProvingTaskAssigned, status)

	assert.NoError(t, batchOrm.RefundAttemptsByHash(context.Background(), "0xbatch-a"))
	active, total, err = batchOrm.GetAttemptsByHash(context.Background(), "0xbatch-a")
	assert.NoError(t, err)
	assert.Equal(t, int16(0), active)
	assert.Equal(t, int16(0), total)
	status, err = batchOrm.GetProvingStatusByHash(context.Background(), "0xbatch-a")
	assert.NoError(t, err)
	assert.Equal(t, types.ProvingTaskUnassigned, status)

	// (b) refund on a verified row -> no-op
	insertBatch(2, "0xbatch-b", int16(types.ProvingTaskVerified), 1, 1)
	assert.NoError(t, batchOrm.RefundAttemptsByHash(context.Background(), "0xbatch-b"))
	active, total, err = batchOrm.GetAttemptsByHash(context.Background(), "0xbatch-b")
	assert.NoError(t, err)
	assert.Equal(t, int16(1), active)
	assert.Equal(t, int16(1), total)
	status, err = batchOrm.GetProvingStatusByHash(context.Background(), "0xbatch-b")
	assert.NoError(t, err)
	assert.Equal(t, types.ProvingTaskVerified, status)

	// (c) refund when attempts are already 0 -> no underflow, row unchanged
	insertBatch(3, "0xbatch-c", int16(types.ProvingTaskAssigned), 0, 0)
	assert.NoError(t, batchOrm.RefundAttemptsByHash(context.Background(), "0xbatch-c"))
	active, total, err = batchOrm.GetAttemptsByHash(context.Background(), "0xbatch-c")
	assert.NoError(t, err)
	assert.Equal(t, int16(0), active)
	assert.Equal(t, int16(0), total)
	status, err = batchOrm.GetProvingStatusByHash(context.Background(), "0xbatch-c")
	assert.NoError(t, err)
	assert.Equal(t, types.ProvingTaskAssigned, status)
}

func TestBundleRefundAttemptsByHash(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	bundleOrm := NewBundle(db)
	insertBundle := func(hash string, provingStatus, totalAttempts, activeAttempts int16) {
		execErr := db.Exec(`INSERT INTO bundle (hash, start_batch_index, end_batch_index, start_batch_hash, end_batch_hash,
			codec_version, proving_status, total_attempts, active_attempts)
			VALUES (?, 1, 2, '0x1', '0x2', 10, ?, ?, ?)`,
			hash, provingStatus, totalAttempts, activeAttempts).Error
		assert.NoError(t, execErr)
	}
	getAttempts := func(hash string) (int16, int16) {
		bundle, getErr := bundleOrm.GetBundleByHash(context.Background(), hash)
		assert.NoError(t, getErr)
		return bundle.ActiveAttempts, bundle.TotalAttempts
	}

	// (a) charge via UpdateBundleAttempts then refund -> both counters back to 0, status back to unassigned
	insertBundle("0xbundle-a", int16(types.ProvingTaskUnassigned), 0, 0)
	rowsAffected, err := bundleOrm.UpdateBundleAttempts(context.Background(), "0xbundle-a", 0, 0)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)
	active, total := getAttempts("0xbundle-a")
	assert.Equal(t, int16(1), active)
	assert.Equal(t, int16(1), total)
	status, err := bundleOrm.GetProvingStatusByHash(context.Background(), "0xbundle-a")
	assert.NoError(t, err)
	assert.Equal(t, types.ProvingTaskAssigned, status)

	assert.NoError(t, bundleOrm.RefundAttemptsByHash(context.Background(), "0xbundle-a"))
	active, total = getAttempts("0xbundle-a")
	assert.Equal(t, int16(0), active)
	assert.Equal(t, int16(0), total)
	status, err = bundleOrm.GetProvingStatusByHash(context.Background(), "0xbundle-a")
	assert.NoError(t, err)
	assert.Equal(t, types.ProvingTaskUnassigned, status)

	// (b) refund on a verified row -> no-op
	insertBundle("0xbundle-b", int16(types.ProvingTaskVerified), 1, 1)
	assert.NoError(t, bundleOrm.RefundAttemptsByHash(context.Background(), "0xbundle-b"))
	active, total = getAttempts("0xbundle-b")
	assert.Equal(t, int16(1), active)
	assert.Equal(t, int16(1), total)
	status, err = bundleOrm.GetProvingStatusByHash(context.Background(), "0xbundle-b")
	assert.NoError(t, err)
	assert.Equal(t, types.ProvingTaskVerified, status)

	// (c) refund when attempts are already 0 -> no underflow, row unchanged
	insertBundle("0xbundle-c", int16(types.ProvingTaskAssigned), 0, 0)
	assert.NoError(t, bundleOrm.RefundAttemptsByHash(context.Background(), "0xbundle-c"))
	active, total = getAttempts("0xbundle-c")
	assert.Equal(t, int16(0), active)
	assert.Equal(t, int16(0), total)
	status, err = bundleOrm.GetProvingStatusByHash(context.Background(), "0xbundle-c")
	assert.NoError(t, err)
	assert.Equal(t, types.ProvingTaskAssigned, status)
}
