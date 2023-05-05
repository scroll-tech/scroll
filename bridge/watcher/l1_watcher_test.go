package watcher

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	geth_types "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"

	commonTypes "scroll-tech/common/types"

	"scroll-tech/database"
	"scroll-tech/database/migrate"
)

func setup(t *testing.T) (*L1WatcherClient, database.OrmFactory) {
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	client, err := ethclient.Dial(base.L1gethImg.Endpoint())
	assert.NoError(t, err)

	l1Cfg := cfg.L1Config

	watcher := NewL1WatcherClient(context.Background(), client, l1Cfg.StartHeight, l1Cfg.Confirmations, l1Cfg.L1MessengerAddress, l1Cfg.L1MessageQueueAddress, l1Cfg.RelayerConfig.RollupContractAddress, db)
	assert.NoError(t, watcher.FetchContractEvent())
	return watcher, db
}

func testStartWatcher(t *testing.T) {
	watcher, _ := setup(t)
	assert.NoError(t, watcher.FetchContractEvent())
}

func testL1WatcherClientFetchBlockHeader(t *testing.T) {
	watcher, db := setup(t)
	convey.Convey("test toBlock < fromBlock", t, func() {
		blockHeight := watcher.ProcessedBlockHeight() - 1
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

	convey.Convey("insert l1 block error", t, func() {
		var c *ethclient.Client
		patchGuard := gomonkey.ApplyMethodFunc(c, "HeaderByNumber", func(ctx context.Context, height *big.Int) (*types.Header, error) {
			height = big.NewInt(100)
			return &types.Header{
				BaseFee: big.NewInt(100),
			}, nil
		})
		defer patchGuard.Reset()

		patchGuard.ApplyMethodFunc(db, "InsertL1Blocks", func(ctx context.Context, blocks []*commonTypes.L1BlockInfo) error {
			return errors.New("insert failed")
		})

		var blockHeight uint64 = 10
		err := watcher.FetchBlockHeader(blockHeight)
		assert.Error(t, err)
	})

	convey.Convey("fetch block header success", t, func() {
		var c *ethclient.Client
		patchGuard := gomonkey.ApplyMethodFunc(c, "HeaderByNumber", func(ctx context.Context, height *big.Int) (*types.Header, error) {
			height = big.NewInt(100)
			return &types.Header{
				BaseFee: big.NewInt(100),
			}, nil
		})
		defer patchGuard.Reset()

		patchGuard.ApplyMethodFunc(db, "InsertL1Blocks", func(ctx context.Context, blocks []*commonTypes.L1BlockInfo) error {
			return nil
		})

		var blockHeight uint64 = 10
		err := watcher.FetchBlockHeader(blockHeight)
		assert.NoError(t, err)
	})
}

func testL1WatcherClientFetchContractEvent(t *testing.T) {
	watcher, db := setup(t)

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
		height = big.NewInt(100)
		return &types.Header{
			Number:  big.NewInt(100),
			BaseFee: big.NewInt(100),
		}, nil
	})
	defer patchGuard.Reset()

	convey.Convey("filter logs failure", t, func() {
		patchGuard.ApplyMethodFunc(c, "FilterLogs", func(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
			return nil, errors.New("call filter failure")
		})
		err := watcher.FetchContractEvent()
		assert.Error(t, err)
		assert.Equal(t, err.Error(), "call filter failure")
	})

	patchGuard.ApplyMethodFunc(c, "FilterLogs", func(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
		t1 := common.Address{}
		t1.SetBytes([]byte("0x0000000000000000000000000000000000000000"))
		return []types.Log{
			{
				Address: t1,
			},
		}, nil
	})

	convey.Convey("parse bridge event logs failure", t, func() {
		targetErr := errors.New("parse log failure")
		patchGuard.ApplyPrivateMethod(watcher, "parseBridgeEventLogs", func(*L1WatcherClient, []geth_types.Log) ([]*commonTypes.L1Message, []relayedMessage, []rollupEvent, error) {
			return nil, nil, nil, targetErr
		})
		err := watcher.FetchContractEvent()
		assert.Error(t, err)
		assert.Equal(t, err.Error(), targetErr.Error())
	})

	patchGuard.ApplyPrivateMethod(watcher, "parseBridgeEventLogs", func(*L1WatcherClient, []geth_types.Log) ([]*commonTypes.L1Message, []relayedMessage, []rollupEvent, error) {
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

		relayedMessageEvents := []relayedMessage{
			{
				msgHash:      common.HexToHash("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347"),
				txHash:       common.HexToHash("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"),
				isSuccessful: true,
			},
			{
				msgHash:      common.HexToHash("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"),
				txHash:       common.HexToHash("0xb4c11951957c6f8f642c4af61cd6b24640fec6dc7fc607ee8206a99e92410d30"),
				isSuccessful: true,
			},
		}
		return nil, relayedMessageEvents, rollupEvents, nil
	})

	convey.Convey("db get rollup status by hash list failure", t, func() {
		targetErr := errors.New("get db failure")
		patchGuard.ApplyMethodFunc(db, "GetRollupStatusByHashList", func(hashes []string) ([]commonTypes.RollupStatus, error) {
			return nil, targetErr
		})
		err := watcher.FetchContractEvent()
		assert.Error(t, err)
		assert.Equal(t, err.Error(), targetErr.Error())
	})

	convey.Convey("rollup status mismatch batch hashes length", t, func() {
		patchGuard.ApplyMethodFunc(db, "GetRollupStatusByHashList", func(hashes []string) ([]commonTypes.RollupStatus, error) {
			s := []commonTypes.RollupStatus{
				commonTypes.RollupFinalized,
			}
			return s, nil
		})
		err := watcher.FetchContractEvent()
		assert.NoError(t, err)
	})

	patchGuard.ApplyMethodFunc(db, "GetRollupStatusByHashList", func(hashes []string) ([]commonTypes.RollupStatus, error) {
		s := []commonTypes.RollupStatus{
			commonTypes.RollupPending,
			commonTypes.RollupCommitting,
		}
		return s, nil
	})

	convey.Convey("db update RollupFinalized status failure", t, func() {
		targetErr := errors.New("UpdateFinalizeTxHashAndRollupStatus RollupFinalized failure")
		patchGuard.ApplyMethodFunc(db, "UpdateFinalizeTxHashAndRollupStatus", func(context.Context, string, string, commonTypes.RollupStatus) error {
			return targetErr
		})
		err := watcher.FetchContractEvent()
		assert.Error(t, err)
		assert.Equal(t, targetErr.Error(), err.Error())
	})

	patchGuard.ApplyMethodFunc(db, "UpdateFinalizeTxHashAndRollupStatus", func(context.Context, string, string, commonTypes.RollupStatus) error {
		return nil
	})

	convey.Convey("db update RollupCommitted status failure", t, func() {
		targetErr := errors.New("UpdateCommitTxHashAndRollupStatus RollupCommitted failure")
		patchGuard.ApplyMethodFunc(db, "UpdateCommitTxHashAndRollupStatus", func(context.Context, string, string, commonTypes.RollupStatus) error {
			return targetErr
		})
		err := watcher.FetchContractEvent()
		assert.Error(t, err)
		assert.Equal(t, targetErr.Error(), err.Error())
	})

	patchGuard.ApplyMethodFunc(db, "UpdateCommitTxHashAndRollupStatus", func(context.Context, string, string, commonTypes.RollupStatus) error {
		return nil
	})

	convey.Convey("db update layer2 status and layer1 hash failure", t, func() {
		targetErr := errors.New("UpdateLayer2StatusAndLayer1Hash failure")
		patchGuard.ApplyMethodFunc(db, "UpdateLayer2StatusAndLayer1Hash", func(context.Context, string, commonTypes.MsgStatus, string) error {
			return targetErr
		})
		err := watcher.FetchContractEvent()
		assert.Error(t, err)
		assert.Equal(t, targetErr.Error(), err.Error())
	})

	patchGuard.ApplyMethodFunc(db, "UpdateLayer2StatusAndLayer1Hash", func(context.Context, string, commonTypes.MsgStatus, string) error {
		return nil
	})

	convey.Convey("db save l1 message failure", t, func() {
		targetErr := errors.New("SaveL1Messages failure")
		patchGuard.ApplyMethodFunc(db, "SaveL1Messages", func(context.Context, []*commonTypes.L1Message) error {
			return targetErr
		})
		err := watcher.FetchContractEvent()
		assert.Error(t, err)
		assert.Equal(t, targetErr.Error(), err.Error())
	})

	patchGuard.ApplyMethodFunc(db, "SaveL1Messages", func(context.Context, []*commonTypes.L1Message) error {
		return nil
	})

	convey.Convey("FetchContractEvent success", t, func() {
		err := watcher.FetchContractEvent()
		assert.NoError(t, err)
	})
}
