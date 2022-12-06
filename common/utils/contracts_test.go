package utils_test

import (
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/math"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/utils"
)

func TestComputeBatchID(t *testing.T) {
	// expected generated using contract:
	// ```
	// // SPDX-License-Identifier: MIT

	// pragma solidity ^0.6.6;

	// contract AAA {
	//     uint256 private constant MAX = ~uint256(0);

	//     function _computeBatchId() public pure returns (bytes32) {
	//         return keccak256(abi.encode(bytes32(0), bytes32(0), MAX));
	//     }
	// }
	// ```

	expected := "0xafe1e714d2cd3ed5b0fa0a04ee95cd564b955ab8661c5665588758b48b66e263"
	actual := utils.ComputeBatchID(common.Hash{}, common.Hash{}, math.MaxBig256)
	assert.Equal(t, expected, actual)

	expected = "0xe05698242b035c0e4d1d58e8ab89507ac7a1403b17fd6a7ea87621a32674ec88"
	actual = utils.ComputeBatchID(
		common.HexToHash("0xfaef7761204f43c4ab2528a65fcc7ec2108709e5ebb646bdce9ce3c8862d3f25"),
		common.HexToHash("0xe3abef08cce4b8a0dcc6b7e4dd11f32863007a86f46c1d136682b5d77bdf0f7a"),
		big.NewInt(77233900))
	assert.Equal(t, expected, actual)
}
