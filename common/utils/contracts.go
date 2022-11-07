package utils

// ComputeBatchID compute an unique hash for a batch using "endBlockHash" & "endBlockHash in last batch"
// & "batch height", following the logic in `_computeBatchId` in contracts/src/L1/rollup/ZKRollup.sol
func ComputeBatchID(endBlockHash string, lastEndBlockHash string, index int64) string {
	return ""
}
