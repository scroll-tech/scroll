package relayer

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"

	bridgeabi "scroll-tech/rollup/abi"
	"scroll-tech/rollup/internal/orm"
)

func TestAssembleDABatch(t *testing.T) {
	calldataHex := "0x9bbaa2ba0000000000000000000000000000000000000000000000000000000000000008146793a7d71663cd87ec9713f72242a3798d5e801050130a3e16efaa09fb803e58af2593dadc8b9fff75a2d27199cb97ec115bade109b8d691a512608ef180eb"
	blobsPath := filepath.Join("../../../testdata", "commit_batches_blobs.json")

	calldata, err := hexutil.Decode(strings.TrimSpace(calldataHex))
	assert.NoErrorf(t, err, "failed to decode calldata: %s", calldataHex)

	blobs, err := loadBlobsFromJSON(blobsPath)
	assert.NoErrorf(t, err, "failed to read blobs: %s", blobsPath)
	assert.NotEmpty(t, blobs, "no blobs provided")

	info, err := parseCommitBatchesCalldata(bridgeabi.ScrollChainABI, calldata)
	assert.NoError(t, err)

	codec, err := encoding.CodecFromVersion(encoding.CodecVersion(info.Version))
	assert.NoErrorf(t, err, "failed to get codec from version %d", info.Version)

	parentBatchHash := info.ParentBatchHash
	index := uint64(113571)

	t.Logf("calldata parsed: version=%d parentBatchHash=%s lastBatchHash=%s blobs=%d", info.Version, info.ParentBatchHash.Hex(), info.LastBatchHash.Hex(), len(blobs))

	fromAddr := common.HexToAddress("0x61d8d3e7f7c656493d1d76aaa1a836cedfcbc27b")
	toAddr := common.HexToAddress("0xba50f5340fb9f3bd074bd638c9be13ecb36e603d")
	l1MessagesWithBlockNumbers := map[uint64][]*types.TransactionData{
		11488527: {
			&types.TransactionData{
				Type:  types.L1MessageTxType,
				Nonce: 1072515,
				Gas:   340000,
				To:    &toAddr,
				Value: (*hexutil.Big)(big.NewInt(0)),
				Data:  "0x8ef1332e00000000000000000000000081f3843af1fbab046b771f0d440c04ebb2b7513f000000000000000000000000cec03800074d0ac0854bf1f34153cc4c8baeeb1e00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000105d8300000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000084f03efa3700000000000000000000000000000000000000000000000000000000000024730000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000000171bdb6e3062daaee1845ba4cb1902169feb5a9b9555a882d45637d3bd29eb83500000000000000000000000000000000000000000000000000000000",
				From:  fromAddr,
			},
		},
		11488622: {
			&types.TransactionData{
				Type:  types.L1MessageTxType,
				Nonce: 1072516,
				Gas:   340000,
				To:    &toAddr,
				Value: (*hexutil.Big)(big.NewInt(0)),
				Data:  "0x8ef1332e00000000000000000000000081f3843af1fbab046b771f0d440c04ebb2b7513f000000000000000000000000cec03800074d0ac0854bf1f34153cc4c8baeeb1e00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000105d8400000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000084f03efa370000000000000000000000000000000000000000000000000000000000002474000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000012aeb01535c1845b689bfce22e53029ec59ec75ea20f660d7c5fcd99f55b75b6900000000000000000000000000000000000000000000000000000000",
				From:  fromAddr,
			},
		},
		11489190: {
			&types.TransactionData{
				Type:  types.L1MessageTxType,
				Nonce: 1072517,
				Gas:   168000,
				To:    &toAddr,
				Value: (*hexutil.Big)(big.NewInt(0)),
				Data:  "0x8ef1332e0000000000000000000000003b1399523f819ea4c4d3e76dddefaf4226c6ba570000000000000000000000003b1399523f819ea4c4d3e76dddefaf4226c6ba5700000000000000000000000000000000000000000000000000000000000027100000000000000000000000000000000000000000000000000000000000105d8500000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000000",
				From:  fromAddr,
			},
		},
	}

	for i, blob := range blobs {
		payload, decErr := codec.DecodeBlob(blob)
		assert.NoErrorf(t, decErr, "blob[%d] decode failed", i)
		if decErr != nil {
			continue
		}

		dbBatch := &orm.Batch{
			Index:           index,
			ParentBatchHash: parentBatchHash.Hex(),
		}

		daBatch, asmErr := assembleDABatchFromPayload(payload, dbBatch, codec, l1MessagesWithBlockNumbers)
		assert.NoErrorf(t, asmErr, "blob[%d] assemble failed", i)
		if asmErr == nil {
			t.Logf("blob[%d] DABatch hash=%s", i, daBatch.Hash().Hex())
		}

		index += 1
		parentBatchHash = daBatch.Hash()
	}
}

func loadBlobsFromJSON(path string) ([]*kzg4844.Blob, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var arr []hexutil.Bytes
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, fmt.Errorf("invalid JSON; expect [\"0x...\"] array: %w", err)
	}

	out := make([]*kzg4844.Blob, 0, len(arr))
	var empty kzg4844.Blob
	want := len(empty)

	for i, b := range arr {
		if len(b) != want {
			return nil, fmt.Errorf("blob[%d] length mismatch: got %d, want %d", i, len(b), want)
		}
		blob := new(kzg4844.Blob)
		copy(blob[:], b)
		out = append(out, blob)
	}
	return out, nil
}
