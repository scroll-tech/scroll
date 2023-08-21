package sender

import (
	"math/big"
	"sync/atomic"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"

	"scroll-tech/common/utils"
)

func (s *Sender) estimateLegacyGas(auth *bind.TransactOpts, contract *common.Address, value *big.Int, input []byte, minGasLimit uint64) (*FeeData, error) {
	gasPrice, err := s.client.SuggestGasPrice(s.ctx)
	if err != nil {
		return nil, err
	}
	gasLimit, err := s.estimateGasLimit(auth, contract, input, gasPrice, nil, nil, value, minGasLimit)
	if err != nil {
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
		return nil, err
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
	gasLimit, err := utils.EstimateGas(s.rpcCli, msg, s.blockNumber)
	if err != nil {
		return 0, err
	}
	// Make sure the gas limit is enough to use.
	gasLimit = (120 * gasLimit) / 100
	if minGasLimit > gasLimit {
		gasLimit = minGasLimit
	}

	gasLimit = gasLimit * 15 / 10 // 50% extra gas to void out of gas error

	return gasLimit, nil
}
