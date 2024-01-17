package sender

import (
	"fmt"
	"math/big"
	"sync/atomic"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
)

func (s *Sender) estimateLegacyGas(auth *bind.TransactOpts, contract *common.Address, value *big.Int, input []byte, fallbackGasLimit uint64) (*FeeData, error) {
	gasPrice, err := s.client.SuggestGasPrice(s.ctx)
	if err != nil {
		log.Error("estimateLegacyGas SuggestGasPrice failure", "error", err)
		return nil, err
	}
	gasLimit, _, err := s.estimateGasLimit(auth, contract, input, gasPrice, nil, nil, value, false)
	if err != nil {
		log.Error("estimateLegacyGas estimateGasLimit failure", "gas price", gasPrice, "from", auth.From.Hex(),
			"nonce", auth.Nonce.Uint64(), "contract address", contract.Hex(), "fallback gas limit", fallbackGasLimit, "error", err)
		if fallbackGasLimit == 0 {
			return nil, err
		}
		gasLimit = fallbackGasLimit
	} else {
		gasLimit = gasLimit * 12 / 10 // 20% extra gas to avoid out of gas error
	}
	return &FeeData{
		gasPrice: gasPrice,
		gasLimit: gasLimit,
	}, nil
}

func (s *Sender) estimateDynamicGas(auth *bind.TransactOpts, contract *common.Address, value *big.Int, input []byte, fallbackGasLimit uint64) (*FeeData, error) {
	gasTipCap, err := s.client.SuggestGasTipCap(s.ctx)
	if err != nil {
		log.Error("estimateDynamicGas SuggestGasTipCap failure", "error", err)
		return nil, err
	}

	baseFee := big.NewInt(0)
	if feeGas := atomic.LoadUint64(&s.baseFeePerGas); feeGas != 0 {
		baseFee.SetUint64(feeGas)
	}
	gasFeeCap := new(big.Int).Add(
		gasTipCap,
		new(big.Int).Mul(baseFee, big.NewInt(2)),
	)
	gasLimit, accessList, err := s.estimateGasLimit(auth, contract, input, nil, gasTipCap, gasFeeCap, value, true)
	if err != nil {
		log.Error("estimateDynamicGas estimateGasLimit failure",
			"from", auth.From.Hex(), "nonce", auth.Nonce.Uint64(), "contract address", contract.Hex(),
			"fallback gas limit", fallbackGasLimit, "error", err)
		if fallbackGasLimit == 0 {
			return nil, err
		}
		gasLimit = fallbackGasLimit
	} else {
		gasLimit = gasLimit * 12 / 10 // 20% extra gas to avoid out of gas error
	}
	feeData := &FeeData{
		gasLimit:  gasLimit,
		gasTipCap: gasTipCap,
		gasFeeCap: gasFeeCap,
	}
	if accessList != nil {
		feeData.accessList = *accessList
	}
	return feeData, nil
}

func (s *Sender) estimateGasLimit(opts *bind.TransactOpts, contract *common.Address, input []byte, gasPrice, gasTipCap, gasFeeCap, value *big.Int, useAccessList bool) (uint64, *types.AccessList, error) {
	msg := ethereum.CallMsg{
		From:      opts.From,
		To:        contract,
		GasPrice:  gasPrice,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Value:     value,
		Data:      input,
	}
	gasLimitWithoutAccessList, err := s.client.EstimateGas(s.ctx, msg)
	if err != nil {
		log.Error("estimateGasLimit EstimateGas failure without access list", "error", err)
		return 0, nil, err
	}

	if !useAccessList {
		return gasLimitWithoutAccessList, nil, nil
	}

	var gasLimitWithAccessList uint64
	accessList, gasUsed, errStr, rpcErr := s.gethClient.CreateAccessList(s.ctx, msg)
	if rpcErr != nil {
		log.Error("CreateAccessList RPC error", "error", rpcErr)
		return gasLimitWithoutAccessList, nil, rpcErr
	}
	if errStr != "" {
		log.Error("CreateAccessList reported error", "error", errStr)
		return gasLimitWithoutAccessList, nil, fmt.Errorf(errStr)
	}

	msg.AccessList = *accessList
	gasLimitWithAccessList, err = s.client.EstimateGas(s.ctx, msg)
	if err != nil {
		log.Error("estimateGasLimit EstimateGas failure with access list", "error", err)
		return gasLimitWithoutAccessList, nil, err
	}

	log.Info("gas", "senderName", s.name, "senderService", s.service, "accessListGasUsed", gasUsed, "gasLimitWithAccessList", gasLimitWithAccessList, "gasLimitWithoutAccessList", gasLimitWithoutAccessList)

	if gasLimitWithAccessList < gasLimitWithoutAccessList {
		return gasLimitWithAccessList, accessList, nil
	}
	return gasLimitWithoutAccessList, nil, nil
}
