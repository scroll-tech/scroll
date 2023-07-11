package cross_msg_test

import (
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"bridge-history-api/cross_msg"
)

func TestMergeIntoList(t *testing.T) {
	headers, err := generateHeaders(64)
	assert.NoError(t, err)
	assert.Equal(t, headers[0].Hash(), headers[1].ParentHash)
	headers2, err := generateHeaders(18)
	assert.NoError(t, err)
	result := cross_msg.MergeAddIntoHeaderList(headers, headers2, 64)
	assert.Equal(t, 64, len(result))
	assert.Equal(t, headers2[len(headers2)-1], result[len(result)-1])
	assert.NotEqual(t, headers[0], result[0])
}

func generateHeaders(amount int) ([]*types.Header, error) {
	headers := make([]*types.Header, amount)

	for i := 0; i < amount; i++ {
		var parentHash common.Hash
		if i > 0 {
			parentHash = headers[i-1].Hash()
		}
		nonce, err := rand.Int(rand.Reader, big.NewInt(1<<63-1))
		if err != nil {
			return nil, err
		}
		difficulty := big.NewInt(131072)

		header := &types.Header{
			ParentHash:  parentHash,
			UncleHash:   types.EmptyUncleHash,
			Coinbase:    common.Address{},
			Root:        common.Hash{},
			TxHash:      types.EmptyRootHash,
			ReceiptHash: types.EmptyRootHash,
			Bloom:       types.Bloom{},
			Difficulty:  difficulty,
			Number:      big.NewInt(int64(i)),
			GasLimit:    5000000,
			GasUsed:     0,
			Time:        uint64(i * 15),
			Extra:       []byte{},
			MixDigest:   common.Hash{},
			Nonce:       types.EncodeNonce(uint64(nonce.Uint64())),
		}
		headers[i] = header
	}

	return headers, nil
}

// TODO: add more test cases
// func TestReorg(t *testing.T)
