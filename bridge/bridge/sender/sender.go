package sender

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/math"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/event"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/bridge/config"
)

const (
	// chainHeadChanSize is the size of channel listening to ChainHeadEvent.
	chainHeadChanSize = 10

	// AccessListTxType type for AccessListTx
	AccessListTxType = "AccessListTx"

	// DynamicFeeTxType type for DynamicFeeTx
	DynamicFeeTxType = "DynamicFeeTx"

	// LegacyTxType type for LegacyTx
	LegacyTxType = "LegacyTx"
)

// DefaultSenderConfig The default config
var DefaultSenderConfig = config.SenderConfig{
	Endpoint:            "",
	EscalateBlocks:      3,
	EscalateMultipleNum: 11,
	EscalateMultipleDen: 10,
	MaxGasPrice:         1000_000_000_000, // this is 1000 gwei
	TxType:              AccessListTxType,
}

// Confirmation struct used to indicate transaction confirmation details
type Confirmation struct {
	Nonce uint64
	Hash  common.Hash
	// @todo add more fields
}

// FeeData fee struct used to estimate gas price
type FeeData struct {
	maxFeePerGas         *big.Int
	maxPriorityFeePerGas *big.Int
	gasPrice             *big.Int
}

// PendingTransaction submitted but pending transactions
type PendingTransaction struct {
	submitAt *big.Int
	tx       *types.Transaction
}

// Sender Transaction sender to send transaction to l1/l2 geth
type Sender struct {
	config  config.SenderConfig
	client  *ethclient.Client // The client to retrieve on chain data or send transaction.
	chainID *big.Int          // The chain id of the endpoint
	ctx     context.Context

	signer       types.Signer      // The transaction Signer object.
	prv          *ecdsa.PrivateKey // The private key of the sender.
	address      common.Address    // The hex40 address of the sender.
	nonce        uint64            // The current nonce of the sender.
	pendingNonce uint64

	blockNumber   *big.Int // Current block number on chain.
	baseFeePerGas *big.Int // Current base fee per gas on chain

	mu sync.Mutex

	pendingTxns map[uint64]PendingTransaction // Mapping from nonce to pending transaction

	chainHeadCh  chan *types.Header
	chainHeadSub event.Subscription

	stop chan bool
}

// NewSender returns a new instance of transaction sender
// txConfirmationCh is used to notify confirmed transaction
func NewSender(ctx context.Context, txConfirmationCh chan<- *Confirmation, config config.SenderConfig, prv *ecdsa.PrivateKey) (*Sender, error) {
	client, err := ethclient.Dial(config.Endpoint)
	if err != nil {
		return nil, err
	}

	// get chainID from client
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, err
	}

	// get block number from client
	block, err := client.BlockByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}

	var signer types.Signer
	if config.TxType != DynamicFeeTxType {
		signer = types.NewEIP2930Signer(chainID)
	} else {
		signer = types.NewLondonSigner(chainID)
	}

	// convert to hex40 address
	address := crypto.PubkeyToAddress(prv.PublicKey)
	log.Info("sender",
		"chainID", chainID.String(),
		"address", address.String(),
	)

	// get current nonce from client
	// @todo should be able to recover nonce from local db
	nonce, err := client.NonceAt(ctx, address, nil)
	if err != nil {
		return nil, err
	}

	sender := &Sender{
		config:        config,
		client:        client,
		chainID:       chainID,
		ctx:           ctx,
		signer:        signer,
		prv:           prv,
		address:       address,
		nonce:         nonce,
		pendingNonce:  nonce,
		blockNumber:   block.Number(),
		baseFeePerGas: block.BaseFee(),
		pendingTxns:   make(map[uint64]PendingTransaction),
		chainHeadCh:   make(chan *types.Header, chainHeadChanSize),
		stop:          make(chan bool),
	}

	go sender.Loop(txConfirmationCh)

	return sender, nil
}

// Stop stop the sender module.
func (s *Sender) Stop() {
	s.stop <- true
	log.Info("Transaction sender stopped")
}

func (sender *Sender) getFeeData() (*FeeData, error) {
	// @todo change it when Scroll enable EIP1559
	if sender.config.TxType != DynamicFeeTxType {
		// estimate gas price
		gasPrice, err := sender.client.SuggestGasPrice(sender.ctx)
		if err != nil {
			return nil, err
		}
		return &FeeData{
			gasPrice: gasPrice,
		}, nil
	}
	gasTipCap, err := sender.client.SuggestGasTipCap(sender.ctx)
	if err != nil {
		return nil, err
	}
	// Make sure feeCap is bigger than txpool's gas price. 1000000000 is l2geth's default pool.gas value.
	maxFeePerGas := math.BigMax(sender.baseFeePerGas, big.NewInt(1000000000))
	return &FeeData{
		maxFeePerGas:         math.BigMax(maxFeePerGas, gasTipCap),
		maxPriorityFeePerGas: math.BigMin(maxFeePerGas, gasTipCap),
	}, nil
}

// SendTransaction send a signed L2tL1 transaction.
func (sender *Sender) SendTransaction(target *common.Address, value *big.Int, data []byte) (uint64, *common.Hash, error) {
	var txData types.TxData

	// estimate gas limit
	call := ethereum.CallMsg{
		From:       sender.address,
		To:         target,
		Gas:        0,
		GasPrice:   nil,
		GasFeeCap:  nil,
		GasTipCap:  nil,
		Value:      value,
		Data:       data,
		AccessList: make(types.AccessList, 0),
	}
	gasLimit, err := sender.client.EstimateGas(sender.ctx, call)
	if err != nil {
		return math.MaxUint64, nil, err
	}
	gasLimit = gasLimit * 15 / 10 // 50% extra gas to void out of gas error

	// estimate gas fee
	feeData, err := sender.getFeeData()
	if err != nil {
		return math.MaxUint64, nil, err
	}

	to := *target // copy it
	// lock here to avoit blocking when call `SuggestGasPrice`
	sender.mu.Lock()
	defer sender.mu.Unlock()
	if sender.config.TxType == LegacyTxType {
		// for ganacha mock node
		txData = &types.LegacyTx{
			Nonce:    sender.nonce,
			GasPrice: feeData.gasPrice,
			Gas:      gasLimit,
			To:       &to,
			Value:    new(big.Int).Set(value),
			Data:     common.CopyBytes(data),
			V:        new(big.Int),
			R:        new(big.Int),
			S:        new(big.Int),
		}
	} else if sender.config.TxType == AccessListTxType {
		txData = &types.AccessListTx{
			ChainID:    sender.chainID,
			Nonce:      sender.nonce,
			GasPrice:   feeData.gasPrice,
			Gas:        gasLimit,
			To:         &to,
			Value:      new(big.Int).Set(value),
			Data:       common.CopyBytes(data),
			AccessList: make(types.AccessList, 0),
			V:          new(big.Int),
			R:          new(big.Int),
			S:          new(big.Int),
		}
	} else {
		txData = &types.DynamicFeeTx{
			Nonce:      sender.nonce,
			To:         &to,
			Data:       common.CopyBytes(data),
			Gas:        gasLimit,
			AccessList: make(types.AccessList, 0),
			Value:      new(big.Int).Set(value),
			ChainID:    sender.chainID,
			GasTipCap:  feeData.maxPriorityFeePerGas,
			GasFeeCap:  feeData.maxFeePerGas,
			V:          new(big.Int),
			R:          new(big.Int),
			S:          new(big.Int),
		}
	}

	tx := types.NewTx(txData)
	tx, err = types.SignTx(tx, sender.signer, sender.prv)
	if err != nil {
		// sign tx failed, this is not likely to happen
		return math.MaxUint64, nil, err
	}

	err = sender.client.SendTransaction(sender.ctx, tx)
	if err != nil {
		log.Error("sender SendTransaction", "nonce", sender.nonce, "to", to, "err", err)
		// send transaction failed
		// @todo there are cases when the transaction submitted but rpc returns with error.
		// then, we will got invalid nonce error in subsequent call, we should handle it properly.

		// use "strings.Contains" instead of "errors.Is(core.ErrNonceTooLow) || errors.Is(core.ErrNonceTooHigh)"
		// for gananche compatibility
		if strings.Contains(err.Error(), "nonce") {
			// Adjust nonce. Since sender keep retrying `SendTransaction`, after the adjustment,
			// sender will `SendTransaction` again with the updated nonce
			nonce, err2 := sender.client.NonceAt(sender.ctx, sender.address, nil)
			if err2 != nil {
				return math.MaxUint64, nil, err2
			}
			sender.nonce = nonce
		}

		return math.MaxUint64, nil, err
	}
	// add pending transaction to queue
	sender.pendingTxns[sender.nonce] = PendingTransaction{
		submitAt: new(big.Int).Set(sender.blockNumber),
		tx:       tx,
	}
	sender.nonce++
	hash := tx.Hash()
	return sender.nonce - 1, &hash, nil
}

func (sender *Sender) resubmitTransaction(tx *types.Transaction) error {
	var txData types.TxData

	// estimate gas fee
	// @todo move query out of lock scope
	feeData, err := sender.getFeeData()
	if err != nil {
		return err
	}

	escalateMultipleNum := new(big.Int).SetUint64(sender.config.EscalateMultipleNum)
	escalateMultipleDen := new(big.Int).SetUint64(sender.config.EscalateMultipleDen)
	maxGasPrice := new(big.Int).SetUint64(sender.config.MaxGasPrice)

	if sender.config.TxType == LegacyTxType {
		// for ganacha mock node
		gasPrice := escalateMultipleNum.Mul(escalateMultipleNum, tx.GasPrice())
		gasPrice = gasPrice.Div(gasPrice, escalateMultipleDen)
		if gasPrice.Cmp(feeData.gasPrice) < 0 {
			gasPrice = feeData.gasPrice
		}
		if gasPrice.Cmp(maxGasPrice) > 0 {
			gasPrice = maxGasPrice
		}
		txData = &types.LegacyTx{
			Nonce:    tx.Nonce(),
			GasPrice: gasPrice,
			Gas:      tx.Gas(),
			To:       tx.To(),
			Value:    tx.Value(),
			Data:     tx.Data(),
			V:        new(big.Int),
			R:        new(big.Int),
			S:        new(big.Int),
		}
	} else if sender.config.TxType == AccessListTxType {
		gasPrice := escalateMultipleNum.Mul(escalateMultipleNum, tx.GasPrice())
		gasPrice = gasPrice.Div(gasPrice, escalateMultipleDen)
		if gasPrice.Cmp(feeData.gasPrice) < 0 {
			gasPrice = feeData.gasPrice
		}
		if gasPrice.Cmp(maxGasPrice) > 0 {
			gasPrice = maxGasPrice
		}
		txData = &types.AccessListTx{
			ChainID:    tx.ChainId(),
			Nonce:      tx.Nonce(),
			GasPrice:   gasPrice,
			Gas:        tx.Gas(),
			To:         tx.To(),
			Value:      tx.Value(),
			Data:       tx.Data(),
			AccessList: make(types.AccessList, 0),
			V:          new(big.Int),
			R:          new(big.Int),
			S:          new(big.Int),
		}
	} else {
		gasTipCap := new(big.Int).Set(tx.GasTipCap())
		gasTipCap = gasTipCap.Mul(gasTipCap, escalateMultipleNum)
		gasTipCap = gasTipCap.Div(gasTipCap, escalateMultipleDen)
		gasFeeCap := new(big.Int).Set(tx.GasFeeCap())
		gasFeeCap = gasFeeCap.Mul(gasFeeCap, escalateMultipleNum)
		gasFeeCap = gasFeeCap.Div(gasFeeCap, escalateMultipleDen)
		if gasFeeCap.Cmp(feeData.maxFeePerGas) < 0 {
			gasFeeCap = feeData.maxFeePerGas
		}
		if gasTipCap.Cmp(feeData.maxPriorityFeePerGas) < 0 {
			gasTipCap = feeData.maxPriorityFeePerGas
		}
		if gasFeeCap.Cmp(maxGasPrice) > 0 {
			gasFeeCap = maxGasPrice
		}
		txData = &types.DynamicFeeTx{
			Nonce:      tx.Nonce(),
			To:         tx.To(),
			Data:       tx.Data(),
			Value:      tx.Value(),
			Gas:        tx.Gas(),
			AccessList: make(types.AccessList, 0),
			ChainID:    tx.ChainId(),
			GasTipCap:  gasTipCap,
			GasFeeCap:  gasFeeCap,
			V:          new(big.Int),
			R:          new(big.Int),
			S:          new(big.Int),
		}
	}

	tx = types.NewTx(txData)
	tx, err = types.SignTx(tx, sender.signer, sender.prv)
	if err != nil {
		// sign tx failed, this is not likely to happen
		return err
	}

	err = sender.client.SendTransaction(sender.ctx, tx)
	if err != nil {
		// send transaction failed
		return err
	}
	// add pending transaction to queue
	sender.pendingTxns[tx.Nonce()] = PendingTransaction{
		submitAt: new(big.Int).Set(sender.blockNumber),
		tx:       tx,
	}
	return nil
}

// CheckPendingTransaction Check pending transaction given number of blocks to wait before confirmation.
func (sender *Sender) CheckPendingTransaction(txConfirmationCh chan<- *Confirmation, escalateBlocks *big.Int) {
	sender.mu.Lock()
	defer sender.mu.Unlock()
	if pending, ok := sender.pendingTxns[sender.pendingNonce]; ok {
		newNonce, err := sender.client.NonceAt(sender.ctx, sender.address, sender.blockNumber)
		if err == nil && newNonce > sender.pendingNonce {
			// transaction confirmed, clear pending txns
			// @todo sync to db
			for {
				if sender.pendingNonce == newNonce {
					return
				}
				delete(sender.pendingTxns, sender.pendingNonce)
				txConfirmationCh <- &Confirmation{
					Nonce: sender.pendingNonce,
					Hash:  pending.tx.Hash(),
				}
				sender.pendingNonce++
			}
		} else {
			// transaction not confirmed after `escalateBlocks` blocks
			// resubmit the transaction with higher nonce
			checkAt := new(big.Int).Set(escalateBlocks)
			checkAt = checkAt.Add(checkAt, pending.submitAt)
			if checkAt.Cmp(sender.blockNumber) < 0 {
				_ = sender.resubmitTransaction(pending.tx)
			}
		}
	}
}

// Loop is the main event loop
func (sender *Sender) Loop(txConfirmationCh chan<- *Confirmation) {
	// trigger by timer
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		var block *types.Block
		var err error
		select {
		case <-sender.stop:
			return
		case <-ticker.C:
			block, err = sender.client.BlockByNumber(sender.ctx, nil)
			if err == nil {
				if block.Number() != nil && block.Number().Cmp(sender.blockNumber) > 0 {
					// update blockNumber and baseFeePerGas
					sender.blockNumber = block.Number()
					sender.baseFeePerGas = block.BaseFee()

					// possible check pending transaction
					sender.CheckPendingTransaction(txConfirmationCh, new(big.Int).SetUint64(sender.config.EscalateBlocks))
				}
			} else {
				log.Warn("BlockByNumber failed", "err", err)
			}
		}
	}
}
