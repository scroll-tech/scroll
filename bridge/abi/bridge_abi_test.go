package bridgeabi_test

import (
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	bridge_abi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/utils"
)

func TestEventSignature(t *testing.T) {
	assert := assert.New(t)

	assert.Equal(bridge_abi.L1SentMessageEventSignature, common.HexToHash("806b28931bc6fbe6c146babfb83d5c2b47e971edb43b4566f010577a0ee7d9f4"))
	assert.Equal(bridge_abi.L1RelayedMessageEventSignature, common.HexToHash("4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c"))
	assert.Equal(bridge_abi.L1FailedRelayedMessageEventSignature, common.HexToHash("99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f"))

	assert.Equal(bridge_abi.L1CommitBatchEventSignature, common.HexToHash("a26d4bd91c4c2eff3b1bf542129607d782506fc1950acfab1472a20d28c06596"))
	assert.Equal(bridge_abi.L1FinalizeBatchEventSignature, common.HexToHash("e20f311a96205960de4d2bb351f7729e5136fa36ae64d7f736c67ddc4ca4cd4b"))

	assert.Equal(bridge_abi.L1AppendMessageEventSignature, common.HexToHash("4e24f8e58edb75fdffd4bd6a38963c5bd49cdf3f7898748e48c58b2076cfe70f"))

	assert.Equal(bridge_abi.L2SentMessageEventSignature, common.HexToHash("806b28931bc6fbe6c146babfb83d5c2b47e971edb43b4566f010577a0ee7d9f4"))
	assert.Equal(bridge_abi.L2RelayedMessageEventSignature, common.HexToHash("4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c"))
	assert.Equal(bridge_abi.L2FailedRelayedMessageEventSignature, common.HexToHash("99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f"))

	assert.Equal(bridge_abi.L2ImportBlockEventSignature, common.HexToHash("fa1488a208a99e5ca060aff7763286188c6a5bdc43964fb76baf67b419450995"))

	assert.Equal(bridge_abi.L2AppendMessageEventSignature, common.HexToHash("faa617c2d8ce12c62637dbce76efcc18dae60574aa95709bdcedce7e76071693"))
}

func TestPackRelayL2MessageWithProof(t *testing.T) {
	assert := assert.New(t)

	l1MessengerABI := bridge_abi.L1MessengerABI

	proof := bridge_abi.IL1ScrollMessengerL2MessageProof{
		BlockHash:        common.Hash{},
		MessageRootProof: make([]common.Hash, 10),
	}
	_, err := l1MessengerABI.Pack("relayMessageWithProof", common.Address{}, common.Address{}, big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), make([]byte, 0), proof)
	assert.NoError(err)
}

func TestPackCommitBatch(t *testing.T) {
	assert := assert.New(t)

	l1RollupABI := bridge_abi.RollupABI

	txns := make([]bridge_abi.IZKRollupLayer2Transaction, 5)
	for i := 0; i < 5; i++ {
		txns[i] = bridge_abi.IZKRollupLayer2Transaction{
			Target:   common.Address{},
			Nonce:    0,
			Gas:      0,
			GasPrice: big.NewInt(0),
			Value:    big.NewInt(0),
			Data:     make([]byte, 0),
			R:        big.NewInt(0),
			S:        big.NewInt(0),
			V:        0,
		}
	}

	header := bridge_abi.IZKRollupLayer2BlockHeader{
		BlockHash:   common.Hash{},
		ParentHash:  common.Hash{},
		BaseFee:     big.NewInt(0),
		StateRoot:   common.Hash{},
		BlockHeight: 0,
		GasUsed:     0,
		Timestamp:   0,
		ExtraData:   make([]byte, 0),
		Txs:         txns,
		MessageRoot: common.Hash{},
	}

	batch := bridge_abi.IZKRollupLayer2Batch{
		BatchIndex: 0,
		ParentHash: common.Hash{},
		Blocks:     []bridge_abi.IZKRollupLayer2BlockHeader{header},
	}

	_, err := l1RollupABI.Pack("commitBatch", batch)
	assert.NoError(err)
}

func TestPackFinalizeBatchWithProof(t *testing.T) {
	assert := assert.New(t)

	l1RollupABI := bridge_abi.RollupABI

	proof := make([]*big.Int, 10)
	instance := make([]*big.Int, 10)
	for i := 0; i < 10; i++ {
		proof[i] = big.NewInt(0)
		instance[i] = big.NewInt(0)
	}

	_, err := l1RollupABI.Pack("finalizeBatchWithProof", common.Hash{}, proof, instance)
	assert.NoError(err)
}

func TestPackRelayL1MessageWithProof(t *testing.T) {
	assert := assert.New(t)

	l2MessengerABI := bridge_abi.L2MessengerABI

	proof := bridge_abi.IL2ScrollMessengerL1MessageProof{
		BlockHash:      common.Hash{},
		StateRootProof: make([]byte, 10),
	}
	_, err := l2MessengerABI.Pack("relayMessageWithProof", common.Address{}, common.Address{}, big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), make([]byte, 0), proof)
	assert.NoError(err)
}

func TestPackImportBlock(t *testing.T) {
	assert := assert.New(t)

	l1BlockContainerABI := bridge_abi.L1BlockContainerABI

	_, err := l1BlockContainerABI.Pack("importBlockHeader", common.Hash{}, make([]byte, 0), make([]byte, 0))
	assert.NoError(err)
}

func TestUnpackL1Event_SentMessage(t *testing.T) {
	log := types.Log{
		Topics: []common.Hash{
			common.HexToHash("0x806b28931bc6fbe6c146babfb83d5c2b47e971edb43b4566f010577a0ee7d9f4"),
			common.HexToHash("0x000000000000000000000000330a32fee6421b9c0b6cfaa6ddaa1ad6c8ed17f9"),
		},
		Data: common.Hex2Bytes("000000000000000000000000f8f8b383285cd23ed2cb9fdf9599b968ddafc3c0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000063c63a8100000000000000000000000000000000000000000000000000000000000000e000000000000000000000000000000000000000000000000000000000000013f1000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000e48431f5c10000000000000000000000001838a36ab91900fa0e637006bb2faa6ef1a5f84100000000000000000000000038ba9a208f34ddc9332f6dfc0e9d567f098958a4000000000000000000000000bf290324852d86976e9982241723e53eaf2d29d0000000000000000000000000bf290324852d86976e9982241723e53eaf2d29d0000000000000000000000000000000000000000000000002b5e3af16b188000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"),
	}

	event := bridge_abi.L1SentMessageEvent{}
	err := utils.UnpackLog(bridge_abi.L1MessengerABI, &event, "SentMessage", log)
	assert.NoError(t, err)
	assert.Equal(t, event.Target, common.HexToAddress("0x330a32fee6421b9c0b6cfaa6ddaa1ad6c8ed17f9"))
	assert.Equal(t, event.Sender, common.HexToAddress("0xf8f8b383285cd23ed2cb9fdf9599b968ddafc3c0"))

	out := make(map[string]interface{})
	err = utils.UnpackLogIntoMap(bridge_abi.L1MessengerABI, out, "SentMessage", log)
	assert.NoError(t, err)
	assert.Equal(t, event.Target, out["target"])
	assert.Equal(t, event.Sender, out["sender"])
	assert.Equal(t, event.Value, out["value"])
	assert.Equal(t, event.Fee, out["fee"])
	assert.Equal(t, event.Deadline, out["deadline"])
	assert.Equal(t, event.Message, out["message"])
	assert.Equal(t, event.MessageNonce, out["messageNonce"])
	assert.Equal(t, event.GasLimit, out["gasLimit"])
}

func TestUnpackL1Event_CommitBatch(t *testing.T) {
	log := types.Log{
		Topics: []common.Hash{
			common.HexToHash("0xa26d4bd91c4c2eff3b1bf542129607d782506fc1950acfab1472a20d28c06596"),
			common.HexToHash("0x214875997226c54175df6dc97e1bce7f3d624263e1b33bf5daf37b3440d8ffbc"),
		},
		Data: common.Hex2Bytes("4476f580301f544e9e76d1c72037fdb53d83e43436639c69d472ad3be0afbdc000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000"),
	}

	event := bridge_abi.L1CommitBatchEvent{}
	err := utils.UnpackLog(bridge_abi.RollupABI, &event, "CommitBatch", log)
	assert.NoError(t, err)
	assert.Equal(t, event.BatchId, common.HexToHash("0x214875997226c54175df6dc97e1bce7f3d624263e1b33bf5daf37b3440d8ffbc"))
	assert.Equal(t, event.BatchHash, common.HexToHash("0x4476f580301f544e9e76d1c72037fdb53d83e43436639c69d472ad3be0afbdc0"))

	out := make(map[string]interface{})
	err = utils.UnpackLogIntoMap(bridge_abi.RollupABI, out, "CommitBatch", log)
	assert.NoError(t, err)
	batchID := out["_batchId"].([32]byte)
	batchHash := out["_batchHash"].([32]byte)
	parentHash := out["_parentHash"].([32]byte)
	assert.Equal(t, event.BatchId, common.BytesToHash(batchID[:]))
	assert.Equal(t, event.BatchHash, common.BytesToHash(batchHash[:]))
	assert.Equal(t, event.BatchIndex, out["_batchIndex"].(*big.Int))
	assert.Equal(t, event.ParentHash, common.BytesToHash(parentHash[:]))
}
