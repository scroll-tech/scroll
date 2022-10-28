package sender

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"math/big"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/math"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/bridge/config"
)

const (

	// AccessListTxType type for AccessListTx
	AccessListTxType = "AccessListTx"

	// DynamicFeeTxType type for DynamicFeeTx
	DynamicFeeTxType = "DynamicFeeTx"

	// LegacyTxType type for LegacyTx
	LegacyTxType = "LegacyTx"
)

var (
	ErrEmptyAccount = errors.New("has no enough accounts to send transaction")
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
	ID           string
	IsSuccessful bool
	TxHash       common.Hash
}

// FeeData fee struct used to estimate gas price
type FeeData struct {
	gasFeeCap *big.Int
	gasTipCap *big.Int
	gasPrice  *big.Int

	gasLimit uint64
}

// PendingTransaction submitted but pending transactions
type PendingTransaction struct {
	submitAt uint64
	id       string
	feeData  *FeeData
	tx       *types.Transaction
}

// Sender Transaction sender to send transaction to l1/l2 geth
type Sender struct {
	config  *config.SenderConfig
	client  *ethclient.Client // The client to retrieve on chain data or send transaction.
	chainID *big.Int          // The chain id of the endpoint
	ctx     context.Context

	// account fields.
	accs *accounts

	mu            sync.Mutex
	blockNumber   uint64   // Current block number on chain.
	baseFeePerGas uint64   // Current base fee per gas on chain
	pendingTxs    sync.Map // Mapping from nonce to pending transaction
	confirmCh     chan *Confirmation

	stopCh chan struct{}
}

// NewSender returns a new instance of transaction sender
// txConfirmationCh is used to notify confirmed transaction
func NewSender(ctx context.Context, config *config.SenderConfig, privs []*ecdsa.PrivateKey) (*Sender, error) {
	if config == nil {
		config = &DefaultSenderConfig
	}
	client, err := ethclient.Dial(config.Endpoint)
	if err != nil {
		return nil, err
	}

	// get chainID from client
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, err
	}

	accs, err := newAccounts(ctx, client, privs)
	if err != nil {
		return nil, fmt.Errorf("failed to create account pool, err: %v", err)
	}

	// get header by number
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}

	sender := &Sender{
		ctx:           ctx,
		config:        config,
		client:        client,
		chainID:       chainID,
		accs:          accs,
		confirmCh:     make(chan *Confirmation, 128),
		baseFeePerGas: header.BaseFee.Uint64(),
		pendingTxs:    sync.Map{},
		stopCh:        make(chan struct{}),
	}

	go sender.loop(ctx)

	return sender, nil
}

// Stop stop the sender module.
func (s *Sender) Stop() {
	close(s.stopCh)
	log.Info("Transaction sender stopped")
}

// ConfirmChan channel used to communicate with transaction sender
func (s *Sender) ConfirmChan() <-chan *Confirmation {
	return s.confirmCh
}

func (s *Sender) AccCount() int {
	return len(s.accs.accounts)
}

func (s *Sender) getFeeData(auth *bind.TransactOpts, target *common.Address, value *big.Int, data []byte) (*FeeData, error) {
	// estimate gas limit
	if data == nil {
		data = []byte{}
	}
	gasLimit, err := s.client.EstimateGas(s.ctx, ethereum.CallMsg{From: auth.From, To: target, Value: value, Data: data})
	if err != nil {
		return nil, err
	}
	gasLimit = gasLimit * 15 / 10 // 50% extra gas to void out of gas error
	// @todo change it when Scroll enable EIP1559
	if s.config.TxType != DynamicFeeTxType {
		// estimate gas price
		var gasPrice *big.Int
		gasPrice, err = s.client.SuggestGasPrice(s.ctx)
		if err != nil {
			return nil, err
		}
		return &FeeData{
			gasPrice: gasPrice,
			gasLimit: gasLimit,
		}, nil
	}
	gasTipCap, err := s.client.SuggestGasTipCap(s.ctx)
	if err != nil {
		return nil, err
	}
	// Make sure feeCap is bigger than txpool's gas price. 1000000000 is l2geth's default pool.gas value.
	baseFee := atomic.LoadUint64(&s.baseFeePerGas)
	maxFeePerGas := math.BigMax(big.NewInt(int64(baseFee)), big.NewInt(1000000000))
	return &FeeData{
		gasFeeCap: math.BigMax(maxFeePerGas, gasTipCap),
		gasTipCap: math.BigMin(maxFeePerGas, gasTipCap),
		gasLimit:  gasLimit,
	}, nil
}

// SendTransaction send a signed L2tL1 transaction.
func (s *Sender) SendTransaction(ID string, target *common.Address, value *big.Int, data []byte) (hash common.Hash, err error) {
	if _, ok := s.pendingTxs.Load(ID); ok {
		return common.Hash{}, fmt.Errorf("has the repeat tx ID, ID: %s", ID)
	}
	// get
	auth := s.accs.getAccount()
	if auth == nil {
		return common.Hash{}, ErrEmptyAccount
	}
	defer s.accs.setAccount(auth)

	var (
		feeData *FeeData
		tx      *types.Transaction
	)
	// estimate gas fee
	if feeData, err = s.getFeeData(auth, target, value, data); err != nil {
		return
	}
	if tx, err = s.createAndSendTx(auth, feeData, target, value, data); err == nil {
		// add pending transaction to queue
		pending := &PendingTransaction{
			tx:       tx,
			id:       ID,
			submitAt: atomic.LoadUint64(&s.blockNumber),
			feeData:  feeData,
		}
		s.pendingTxs.Store(ID, pending)
		return tx.Hash(), nil
	}

	return
}

func (s *Sender) createAndSendTx(auth *bind.TransactOpts, feeData *FeeData, target *common.Address, value *big.Int, data []byte) (tx *types.Transaction, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var (
		nonce  = auth.Nonce.Uint64()
		txData types.TxData
	)
	// lock here to avoit blocking when call `SuggestGasPrice`
	switch s.config.TxType {
	case LegacyTxType:
		// for ganache mock node
		txData = &types.LegacyTx{
			Nonce:    nonce,
			GasPrice: feeData.gasPrice,
			Gas:      feeData.gasLimit,
			To:       target,
			Value:    new(big.Int).Set(value),
			Data:     common.CopyBytes(data),
			V:        new(big.Int),
			R:        new(big.Int),
			S:        new(big.Int),
		}
	case AccessListTxType:
		txData = &types.AccessListTx{
			ChainID:    s.chainID,
			Nonce:      nonce,
			GasPrice:   feeData.gasPrice,
			Gas:        feeData.gasLimit,
			To:         target,
			Value:      new(big.Int).Set(value),
			Data:       common.CopyBytes(data),
			AccessList: make(types.AccessList, 0),
			V:          new(big.Int),
			R:          new(big.Int),
			S:          new(big.Int),
		}
	default:
		txData = &types.DynamicFeeTx{
			Nonce:      nonce,
			To:         target,
			Data:       common.CopyBytes(data),
			Gas:        feeData.gasLimit,
			AccessList: make(types.AccessList, 0),
			Value:      new(big.Int).Set(value),
			ChainID:    s.chainID,
			GasTipCap:  feeData.gasTipCap,
			GasFeeCap:  feeData.gasFeeCap,
			V:          new(big.Int),
			R:          new(big.Int),
			S:          new(big.Int),
		}
	}

	// sign and send
	tx, err = auth.Signer(auth.From, types.NewTx(txData))
	if err != nil {
		log.Error("failed to sign tx", "err", err)
		return
	}
	if err = s.client.SendTransaction(s.ctx, tx); err != nil {
		log.Error("failed to send tx", "tx hash", tx.Hash().String(), "err", err)
		// Check if contain nonce, and reset nonce
		if strings.Contains(err.Error(), "nonce") {
			s.accs.reSetNonce(context.Background(), auth)
		}
		return
	}

	// update nonce
	auth.Nonce = big.NewInt(int64(nonce + 1))
	return
}

func (s *Sender) resubmitTransaction(feeData *FeeData, tx *types.Transaction) (*types.Transaction, error) {
	// Get a idle account from account pool.
	auth := s.accs.getAccount()
	if auth == nil {
		return nil, ErrEmptyAccount
	}
	defer s.accs.setAccount(auth)

	escalateMultipleNum := new(big.Int).SetUint64(s.config.EscalateMultipleNum)
	escalateMultipleDen := new(big.Int).SetUint64(s.config.EscalateMultipleDen)
	maxGasPrice := new(big.Int).SetUint64(s.config.MaxGasPrice)

	switch s.config.TxType {
	case LegacyTxType, AccessListTxType: // `LegacyTxType`is for ganache mock node
		gasPrice := escalateMultipleNum.Mul(escalateMultipleNum, big.NewInt(feeData.gasPrice.Int64()))
		gasPrice = gasPrice.Div(gasPrice, escalateMultipleDen)
		if gasPrice.Cmp(feeData.gasPrice) < 0 {
			gasPrice = feeData.gasPrice
		}
		if gasPrice.Cmp(maxGasPrice) > 0 {
			gasPrice = maxGasPrice
		}
		feeData.gasPrice = gasPrice
	default:
		gasTipCap := big.NewInt(feeData.gasTipCap.Int64())
		gasTipCap = gasTipCap.Mul(gasTipCap, escalateMultipleNum)
		gasTipCap = gasTipCap.Div(gasTipCap, escalateMultipleDen)
		gasFeeCap := big.NewInt(feeData.gasFeeCap.Int64())
		gasFeeCap = gasFeeCap.Mul(gasFeeCap, escalateMultipleNum)
		gasFeeCap = gasFeeCap.Div(gasFeeCap, escalateMultipleDen)
		if gasFeeCap.Cmp(feeData.gasFeeCap) < 0 {
			gasFeeCap = feeData.gasFeeCap
		}
		if gasTipCap.Cmp(feeData.gasTipCap) < 0 {
			gasTipCap = feeData.gasTipCap
		}
		if gasFeeCap.Cmp(maxGasPrice) > 0 {
			gasFeeCap = maxGasPrice
		}
		feeData.gasFeeCap = gasFeeCap
		feeData.gasTipCap = gasTipCap
	}

	return s.createAndSendTx(auth, feeData, tx.To(), tx.Value(), tx.Data())
}

// CheckPendingTransaction Check pending transaction given number of blocks to wait before confirmation.
func (s *Sender) CheckPendingTransaction(header *types.Header) {
	number := header.Number.Uint64()
	atomic.StoreUint64(&s.blockNumber, number)
	atomic.StoreUint64(&s.baseFeePerGas, header.BaseFee.Uint64())
	s.pendingTxs.Range(func(key, value interface{}) bool {
		pending := value.(*PendingTransaction)
		receipt, err := s.client.TransactionReceipt(s.ctx, pending.tx.Hash())
		if (err == nil) && (receipt != nil) {
			if number >= receipt.BlockNumber.Uint64()+s.config.Confirmations {
				s.pendingTxs.Delete(key)
				// send confirm message
				s.confirmCh <- &Confirmation{
					ID:           pending.id,
					IsSuccessful: receipt.Status == types.ReceiptStatusSuccessful,
					TxHash:       pending.tx.Hash(),
				}
			}
		} else if s.config.EscalateBlocks+pending.submitAt < number {
			var tx *types.Transaction
			tx, err := s.resubmitTransaction(pending.feeData, pending.tx)
			if err != nil {
				// If accounts channel is empty, wait 1 second.
				if errors.Is(err, ErrEmptyAccount) {
					time.Sleep(time.Second)
				}
				log.Error("failed to resubmit transaction, reset submitAt", "tx hash", pending.tx.Hash().String(), "err", err)
			} else {
				// flush submitAt
				pending.tx = tx
				pending.submitAt = number
			}
		}
		return true
	})
}

// Loop is the main event loop
func (s *Sender) loop(ctx context.Context) {
	checkTick := time.NewTicker(time.Duration(s.config.CheckPendingTime) * time.Second)
	defer checkTick.Stop()

	tick := time.NewTicker(time.Minute * 10)
	defer tick.Stop()

	for {
		select {
		case <-checkTick.C:
			header, err := s.client.HeaderByNumber(s.ctx, nil)
			if err != nil {
				log.Error("failed to get latest head", "err", err)
				continue
			}
			s.CheckPendingTransaction(header)
		case <-tick.C:
			// Check and set balance.
			s.accs.checkAndSetBalance(ctx)
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		}
	}
}
