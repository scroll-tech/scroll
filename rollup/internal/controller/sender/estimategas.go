package sender

import (
	"math/big"
	"sync/atomic"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
)

func (s *Sender) estimateLegacyGas(auth *bind.TransactOpts, contract *common.Address, value *big.Int, input []byte, minGasLimit uint64) (*FeeData, error) {
	gasPrice, err := s.client.SuggestGasPrice(s.ctx)
	if err != nil {
		log.Error("estimateLegacyGas SuggestGasPrice failure", "error", err)
		return nil, err
	}
	gasLimit, err := s.estimateGasLimit(auth, contract, input, gasPrice, nil, nil, value, minGasLimit)
	if err != nil {
		log.Error("estimateLegacyGas estimateGasLimit failure", "gasPrice", gasPrice, "error", err)
		return nil, err
	}
	return &FeeData{
		gasPrice: gasPrice,
		gasLimit: gasLimit,
	}, nil
}

func (s *Sender) estimateDynamicGas(auth *bind.TransactOpts, contract *common.Address, value *big.Int, input []byte, minGasLimit uint64) (*FeeData, error) {
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
	gasLimit, err := s.estimateGasLimit(auth, contract, input, nil, gasTipCap, gasFeeCap, value, minGasLimit)
	if err != nil {
		log.Error("estimateDynamicGas estimateGasLimit failure", "error", err)
		if minGasLimit == 0 {
			return nil, err
		}
		gasLimit = minGasLimit
	}
	return &FeeData{
		gasLimit:  gasLimit,
		gasTipCap: gasTipCap,
		gasFeeCap: gasFeeCap,
	}, nil
}

func (s *Sender) estimateGasLimit(opts *bind.TransactOpts, contract *common.Address, input []byte, gasPrice, gasTipCap, gasFeeCap, value *big.Int, minGasLimit uint64) (uint64, error) {
	msg := ethereum.CallMsg{
		From:      opts.From,
		To:        contract,
		GasPrice:  gasPrice,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Value:     value,
		Data:      input,
	}
	gasLimit, err := s.client.EstimateGas(s.ctx, msg)
	if err != nil {
		log.Error("estimateGasLimit EstimateGas failure", "error", err)
		return 0, err
	}
	if minGasLimit > gasLimit {
		gasLimit = minGasLimit
	}

	gasLimit = gasLimit * 12 / 10 // 20% extra gas to avoid out of gas error

	return gasLimit, nil
}
