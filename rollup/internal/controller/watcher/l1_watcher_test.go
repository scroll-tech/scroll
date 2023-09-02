package watcher

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"scroll-tech/common/database"
	commonTypes "scroll-tech/common/types"

	bridgeAbi "scroll-tech/rollup/abi"
	"scroll-tech/rollup/internal/orm"
	"scroll-tech/rollup/internal/utils"
)

func setupL1Watcher(t *testing.T) (*L1WatcherClient, *gorm.DB) {
	db := setupDB(t)
	client, err := ethclient.Dial(base.L1gethImg.Endpoint())
	assert.NoError(t, err)
	l1Cfg := cfg.L1Config
	watcher := NewL1WatcherClient(context.Background(), client, l1Cfg.StartHeight, l1Cfg.Confirmations, l1Cfg.L1MessageQueueAddress, l1Cfg.RelayerConfig.RollupContractAddress, db, nil)
	assert.NoError(t, watcher.FetchContractEvent())
	return watcher, db
}

func testFetchContractEvent(t *testing.T) {
	watcher, db := setupL1Watcher(t)
	defer database.CloseDB(db)
	assert.NoError(t, watcher.FetchContractEvent())
}

func testL1WatcherClientFetchBlockHeader(t *testing.T) {
	watcher, db := setupL1Watcher(t)
	defer database.CloseDB(db)
	convey.Convey("test toBlock < fromBlock", t, func() {
		var blockHeight uint64
		if watcher.ProcessedBlockHeight() <= 0 {
			blockHeight = 0
		} else {
			blockHeight = watcher.ProcessedBlockHeight() - 1
		}
		err := watcher.FetchBlockHeader(blockHeight)
		assert.NoError(t, err)
	})

	convey.Convey("test get header from client error", t, func() {
		var c *ethclient.Client
		patchGuard := gomonkey.ApplyMethodFunc(c, "HeaderByNumber", func(ctx context.Context, height *big.Int) (*types.Header, error) {
			return nil, ethereum.NotFound
		})
		defer patchGuard.Reset()

		var blockHeight uint64 = 10
		err := watcher.FetchBlockHeader(blockHeight)
		assert.Error(t, err)
	})

	var l1BlockOrm *orm.L1Block
	convey.Convey("insert l1 block error", t, func() {
		var c *ethclient.Client
		patchGuard := gomonkey.ApplyMethodFunc(c, "HeaderByNumber", func(ctx context.Context, height *big.Int) (*types.Header, error) {
			if height == nil {
				height = big.NewInt(100)
			}
			t.Log(height)
			return &types.Header{
				BaseFee: big.NewInt(100),
			}, nil
		})
		defer patchGuard.Reset()

		patchGuard.ApplyMethodFunc(l1BlockOrm, "InsertL1Blocks", func(ctx context.Context, blocks []orm.L1Block) error {
			return errors.New("insert failed")
		})

		var blockHeight uint64 = 10
		err := watcher.FetchBlockHeader(blockHeight)
		assert.Error(t, err)
	})

	convey.Convey("fetch block header success", t, func() {
		var c *ethclient.Client
		patchGuard := gomonkey.ApplyMethodFunc(c, "HeaderByNumber", func(ctx context.Context, height *big.Int) (*types.Header, error) {
			if height == nil {
				height = big.NewInt(100)
			}
			t.Log(height)
			return &types.Header{
				BaseFee: big.NewInt(100),
			}, nil
		})
		defer patchGuard.Reset()

		patchGuard.ApplyMethodFunc(l1BlockOrm, "InsertL1Blocks", func(ctx context.Context, blocks []orm.L1Block) error {
			return nil
		})

		var blockHeight uint64 = 10
		err := watcher.FetchBlockHeader(blockHeight)
		assert.NoError(t, err)
	})
}

func testL1WatcherClientFetchContractEvent(t *testing.T) {
	watcher, db := setupL1Watcher(t)
	defer database.CloseDB(db)

	watcher.SetConfirmations(rpc.SafeBlockNumber)
	convey.Convey("get latest confirmed block number failure", t, func() {
		var c *ethclient.Client
		patchGuard := gomonkey.ApplyMethodFunc(c, "HeaderByNumber", func(ctx context.Context, height *big.Int) (*types.Header, error) {
			return nil, ethereum.NotFound
		})
		defer patchGuard.Reset()
		err := watcher.FetchContractEvent()
		assert.Error(t, err)
	})

	var c *ethclient.Client
	patchGuard := gomonkey.ApplyMethodFunc(c, "HeaderByNumber", func(ctx context.Context, height *big.Int) (*types.Header, error) {
		if height == nil {
			height = big.NewInt(100)
		}
		t.Log(height)
		return &types.Header{
			Number:  big.NewInt(100),
			BaseFee: big.NewInt(100),
		}, nil
	})
	defer patchGuard.Reset()

	convey.Convey("filter logs failure", t, func() {
		targetErr := errors.New("call filter failure")
		patchGuard.ApplyMethodFunc(c, "FilterLogs", func(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
			return nil, targetErr
		})
		err := watcher.FetchContractEvent()
		assert.EqualError(t, err, targetErr.Error())
	})

	patchGuard.ApplyMethodFunc(c, "FilterLogs", func(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
		return []types.Log{
			{
				Address: common.HexToAddress("0x0000000000000000000000000000000000000000"),
			},
		}, nil
	})

	convey.Convey("parse bridge event logs failure", t, func() {
		targetErr := errors.New("parse log failure")
		patchGuard.ApplyPrivateMethod(watcher, "parseBridgeEventLogs", func(*L1WatcherClient, []types.Log) ([]*orm.L1Message, []rollupEvent, error) {
			return nil, nil, targetErr
		})
		err := watcher.FetchContractEvent()
		assert.EqualError(t, err, targetErr.Error())
	})

	patchGuard.ApplyPrivateMethod(watcher, "parseBridgeEventLogs", func(*L1WatcherClient, []types.Log) ([]*orm.L1Message, []rollupEvent, error) {
		rollupEvents := []rollupEvent{
			{
				batchHash: common.HexToHash("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347"),
				txHash:    common.HexToHash("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"),
				status:    commonTypes.RollupFinalized,
			},
			{
				batchHash: common.HexToHash("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"),
				txHash:    common.HexToHash("0xb4c11951957c6f8f642c4af61cd6b24640fec6dc7fc607ee8206a99e92410d30"),
				status:    commonTypes.RollupCommitted,
			},
		}
		return nil, rollupEvents, nil
	})

	var batchOrm *orm.Batch
	convey.Convey("db get rollup status by hash list failure", t, func() {
		targetErr := errors.New("get db failure")
		patchGuard.ApplyMethodFunc(batchOrm, "GetRollupStatusByHashList", func(context.Context, []string) ([]commonTypes.RollupStatus, error) {
			return nil, targetErr
		})
		err := watcher.FetchContractEvent()
		assert.Equal(t, err.Error(), targetErr.Error())
	})

	convey.Convey("rollup status mismatch batch hashes length", t, func() {
		patchGuard.ApplyMethodFunc(batchOrm, "GetRollupStatusByHashList", func(context.Context, []string) ([]commonTypes.RollupStatus, error) {
			s := []commonTypes.RollupStatus{
				commonTypes.RollupFinalized,
			}
			return s, nil
		})
		err := watcher.FetchContractEvent()
		assert.NoError(t, err)
	})

	patchGuard.ApplyMethodFunc(batchOrm, "GetRollupStatusByHashList", func(context.Context, []string) ([]commonTypes.RollupStatus, error) {
		s := []commonTypes.RollupStatus{
			commonTypes.RollupPending,
			commonTypes.RollupCommitting,
		}
		return s, nil
	})

	convey.Convey("db update RollupFinalized status failure", t, func() {
		targetErr := errors.New("UpdateFinalizeTxHashAndRollupStatus RollupFinalized failure")
		patchGuard.ApplyMethodFunc(batchOrm, "UpdateFinalizeTxHashAndRollupStatus", func(context.Context, string, string, commonTypes.RollupStatus) error {
			return targetErr
		})
		err := watcher.FetchContractEvent()
		assert.Equal(t, targetErr.Error(), err.Error())
	})

	patchGuard.ApplyMethodFunc(batchOrm, "UpdateFinalizeTxHashAndRollupStatus", func(context.Context, string, string, commonTypes.RollupStatus) error {
		return nil
	})

	convey.Convey("db update RollupCommitted status failure", t, func() {
		targetErr := errors.New("UpdateCommitTxHashAndRollupStatus RollupCommitted failure")
		patchGuard.ApplyMethodFunc(batchOrm, "UpdateCommitTxHashAndRollupStatus", func(context.Context, string, string, commonTypes.RollupStatus) error {
			return targetErr
		})
		err := watcher.FetchContractEvent()
		assert.Equal(t, targetErr.Error(), err.Error())
	})

	patchGuard.ApplyMethodFunc(batchOrm, "UpdateCommitTxHashAndRollupStatus", func(context.Context, string, string, commonTypes.RollupStatus) error {
		return nil
	})

	var l1MessageOrm *orm.L1Message
	convey.Convey("db save l1 message failure", t, func() {
		targetErr := errors.New("SaveL1Messages failure")
		patchGuard.ApplyMethodFunc(l1MessageOrm, "SaveL1Messages", func(context.Context, []*orm.L1Message) error {
			return targetErr
		})
		err := watcher.FetchContractEvent()
		assert.Equal(t, targetErr.Error(), err.Error())
	})

	patchGuard.ApplyMethodFunc(l1MessageOrm, "SaveL1Messages", func(context.Context, []*orm.L1Message) error {
		return nil
	})

	convey.Convey("FetchContractEvent success", t, func() {
		err := watcher.FetchContractEvent()
		assert.NoError(t, err)
	})
}

func testParseBridgeEventLogsL1QueueTransactionEventSignature(t *testing.T) {
	watcher, db := setupL1Watcher(t)
	defer database.CloseDB(db)

	logs := []types.Log{
		{
			Topics:      []common.Hash{bridgeAbi.L1QueueTransactionEventSignature},
			BlockNumber: 100,
			TxHash:      common.HexToHash("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347"),
		},
	}

	convey.Convey("unpack QueueTransaction log failure", t, func() {
		targetErr := errors.New("UnpackLog QueueTransaction failure")
		patchGuard := gomonkey.ApplyFunc(utils.UnpackLog, func(c *abi.ABI, out interface{}, event string, log types.Log) error {
			return targetErr
		})
		defer patchGuard.Reset()

		l2Messages, rollupEvents, err := watcher.parseBridgeEventLogs(logs)
		assert.EqualError(t, err, targetErr.Error())
		assert.Empty(t, l2Messages)
		assert.Empty(t, rollupEvents)
	})

	convey.Convey("L1QueueTransactionEventSignature success", t, func() {
		patchGuard := gomonkey.ApplyFunc(utils.UnpackLog, func(c *abi.ABI, out interface{}, event string, log types.Log) error {
			tmpOut := out.(*bridgeAbi.L1QueueTransactionEvent)
			tmpOut.QueueIndex = 100
			tmpOut.Data = []byte("test data")
			tmpOut.Sender = common.HexToAddress("0xb4c11951957c6f8f642c4af61cd6b24640fec6dc7fc607ee8206a99e92410d30")
			tmpOut.Value = big.NewInt(1000)
			tmpOut.Target = common.HexToAddress("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5")
			tmpOut.GasLimit = big.NewInt(10)
			return nil
		})
		defer patchGuard.Reset()

		l2Messages, rollupEvents, err := watcher.parseBridgeEventLogs(logs)
		assert.NoError(t, err)
		assert.Empty(t, rollupEvents)
		assert.Len(t, l2Messages, 1)
		assert.Equal(t, l2Messages[0].Value, big.NewInt(1000).String())
	})
}

func testParseBridgeEventLogsL1CommitBatchEventSignature(t *testing.T) {
	watcher, db := setupL1Watcher(t)
	defer database.CloseDB(db)
	logs := []types.Log{
		{
			Topics:      []common.Hash{bridgeAbi.L1CommitBatchEventSignature},
			BlockNumber: 100,
			TxHash:      common.HexToHash("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347"),
		},
	}

	convey.Convey("unpack CommitBatch log failure", t, func() {
		targetErr := errors.New("UnpackLog CommitBatch failure")
		patchGuard := gomonkey.ApplyFunc(utils.UnpackLog, func(c *abi.ABI, out interface{}, event string, log types.Log) error {
			return targetErr
		})
		defer patchGuard.Reset()

		l2Messages, rollupEvents, err := watcher.parseBridgeEventLogs(logs)
		assert.EqualError(t, err, targetErr.Error())
		assert.Empty(t, l2Messages)
		assert.Empty(t, rollupEvents)
	})

	convey.Convey("L1CommitBatchEventSignature success", t, func() {
		msgHash := common.HexToHash("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5")
		patchGuard := gomonkey.ApplyFunc(utils.UnpackLog, func(c *abi.ABI, out interface{}, event string, log types.Log) error {
			tmpOut := out.(*bridgeAbi.L1CommitBatchEvent)
			tmpOut.BatchHash = msgHash
			return nil
		})
		defer patchGuard.Reset()

		l2Messages, rollupEvents, err := watcher.parseBridgeEventLogs(logs)
		assert.NoError(t, err)
		assert.Empty(t, l2Messages)
		assert.Len(t, rollupEvents, 1)
		assert.Equal(t, rollupEvents[0].batchHash, msgHash)
		assert.Equal(t, rollupEvents[0].status, commonTypes.RollupCommitted)
	})
}

func testParseBridgeEventLogsL1FinalizeBatchEventSignature(t *testing.T) {
	watcher, db := setupL1Watcher(t)
	defer database.CloseDB(db)
	logs := []types.Log{
		{
			Topics:      []common.Hash{bridgeAbi.L1FinalizeBatchEventSignature},
			BlockNumber: 100,
			TxHash:      common.HexToHash("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347"),
		},
	}

	convey.Convey("unpack FinalizeBatch log failure", t, func() {
		targetErr := errors.New("UnpackLog FinalizeBatch failure")
		patchGuard := gomonkey.ApplyFunc(utils.UnpackLog, func(c *abi.ABI, out interface{}, event string, log types.Log) error {
			return targetErr
		})
		defer patchGuard.Reset()

		l2Messages, rollupEvents, err := watcher.parseBridgeEventLogs(logs)
		assert.EqualError(t, err, targetErr.Error())
		assert.Empty(t, l2Messages)
		assert.Empty(t, rollupEvents)
	})

	convey.Convey("L1FinalizeBatchEventSignature success", t, func() {
		msgHash := common.HexToHash("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5")
		patchGuard := gomonkey.ApplyFunc(utils.UnpackLog, func(c *abi.ABI, out interface{}, event string, log types.Log) error {
			tmpOut := out.(*bridgeAbi.L1FinalizeBatchEvent)
			tmpOut.BatchHash = msgHash
			return nil
		})
		defer patchGuard.Reset()

		l2Messages, rollupEvents, err := watcher.parseBridgeEventLogs(logs)
		assert.NoError(t, err)
		assert.Empty(t, l2Messages)
		assert.Len(t, rollupEvents, 1)
		assert.Equal(t, rollupEvents[0].batchHash, msgHash)
		assert.Equal(t, rollupEvents[0].status, commonTypes.RollupFinalized)
	})
}
