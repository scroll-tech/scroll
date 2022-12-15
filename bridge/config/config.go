package config

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"scroll-tech/common/utils"

	"scroll-tech/database"

	apollo_config "scroll-tech/common/apollo"
)

// Config load configuration items.
type Config struct {
	L1Config *L1Config          `json:"l1_config"`
	L2Config *L2Config          `json:"l2_config"`
	DBConfig *database.DBConfig `json:"db_config"`
}

// NewConfig returns a new instance of Config.
func NewConfig(file string) (*Config, error) {
	buf, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	err = json.Unmarshal(buf, cfg)
	if err != nil {
		return nil, err
	}

	// cover value by env fields
	cfg.DBConfig.DSN = utils.GetEnvWithDefault("DB_DSN", cfg.DBConfig.DSN)
	cfg.DBConfig.DriverName = utils.GetEnvWithDefault("DB_DRIVER", cfg.DBConfig.DriverName)

	return cfg, nil
}

// GetMaxGasPrice : get the maximum gas price can be used to send transaction.
func GetMaxGasPrice() uint64 {
	maxGasPriceStr := apollo_config.AgolloClient.GetStringValue("maxGasPrice", "10000000000")
	maxGasPrice, err := strconv.ParseInt(maxGasPriceStr, 10, 64)
	if err != nil {
		return 10000000000
	}
	return uint64(maxGasPrice)
}

// GetEscalateMultipleNum : get the numerator of gas price escalate multiple.
func GetEscalateMultipleNum() uint64 {
	return uint64(apollo_config.AgolloClient.GetIntValue("escalateMultipleNum", 11))
}

// GetEscalateMultipleDen : get the denominator of gas price escalate multiple.
func GetEscalateMultipleDen() uint64 {
	return uint64(apollo_config.AgolloClient.GetIntValue("escalateMultipleDen", 10))
}

// GetEscalateBlocks : get the number of blocks to wait to escalate increase gas price of the transaction.
func GetEscalateBlocks() uint64 {
	return uint64(apollo_config.AgolloClient.GetIntValue("escalateBlocks", 100))
}

// GetMinBalance : get the min balance set for check and set balance for sender's accounts.
func GetMinBalance() *big.Int {
	minBalanceStr := apollo_config.AgolloClient.GetStringValue("minBalance", "100000000000000000000")
	minBalance, ok := new(big.Int).SetString(minBalanceStr, 10)
	if ok {
		return minBalance
	}
	minBalance.SetString("100000000000000000000", 10)
	return minBalance
}

// GetL1Confirmations : get l1 block height confirmations number.
func GetL1Confirmations() uint64 {
	return uint64(apollo_config.AgolloClient.GetIntValue("l1Confirmations", 6))
}

// GetL2Confirmations : get l2 block height confirmations number.
func GetL2Confirmations() uint64 {
	return uint64(apollo_config.AgolloClient.GetIntValue("l2Confirmations", 1))
}

// GetL1ContractEventsBlocksFetchLimit : get l1 contract events block fetch limit.
func GetL1ContractEventsBlocksFetchLimit() int64 {
	return int64(apollo_config.AgolloClient.GetIntValue("l1ContractEventsBlocksFetchLimit", 10))
}

// GetL2ContractEventsBlocksFetchLimit : get l2 contract events block fetch limit.
func GetL2ContractEventsBlocksFetchLimit() int64 {
	return int64(apollo_config.AgolloClient.GetIntValue("l2ContractEventsBlocksFetchLimit", 10))
}

// GetL1CheckPendingTime : get l1 check pending time (l2 sender).
func GetL1CheckPendingTime() uint64 {
	return uint64(apollo_config.AgolloClient.GetIntValue("l1CheckPendingTime", 10))
}

// GetL2CheckPendingTime : get l2 check pending time (l1 sender).
func GetL2CheckPendingTime() uint64 {
	return uint64(apollo_config.AgolloClient.GetIntValue("l2CheckPendingTime", 3))
}

// GetL2BlockTracesFetchLimit : get l2 block traces fetch limit.
func GetL2BlockTracesFetchLimit() uint64 {
	return uint64(apollo_config.AgolloClient.GetIntValue("l2BlockTracesFetchLimit", 10))
}

// GetSkippedOpcodes : get skipped opcodes.
func GetSkippedOpcodes() map[string]struct{} {
	skippedOpcodesStr := apollo_config.AgolloClient.GetStringValue("skippedOpcodes", "CREATE2,DELEGATECALL")
	skippedOpcodesSlice := strings.Split(skippedOpcodesStr, ",")
	skippedOpcodes := make(map[string]struct{})
	for _, op := range skippedOpcodesSlice {
		op = strings.TrimSpace(op)
		skippedOpcodes[op] = struct{}{}
	}
	return skippedOpcodes
}
