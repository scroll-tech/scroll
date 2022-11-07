package utils_test

import (
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
}
