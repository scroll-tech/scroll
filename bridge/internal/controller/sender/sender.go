package sender

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync/atomic"
	"time"

	cmapV2 "github.com/orcaman/concurrent-map/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/bridge/internal/config"
	"scroll-tech/bridge/internal/utils"
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
	// ErrNoAvailableAccount indicates no available account error in the account pool.
	ErrNoAvailableAccount = errors.New("sender has no available account to send transaction")
	// ErrFullPending sender's pending pool is full.
	ErrFullPending = errors.New("sender's pending pool is full")
)

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
	signer   *bind.TransactOpts
	tx       *types.Transaction
}

// Sender Transaction sender to send transaction to l1/l2 geth
type Sender struct {
	config  *config.SenderConfig
	client  *ethclient.Client // The client to retrieve on chain data or send transaction.
	chainID *big.Int          // The chain id of the endpoint
	ctx     context.Context
	service string
	name    string

	auth       *bind.TransactOpts
	minBalance *big.Int

	blockNumber   uint64                                            // Current block number on chain.
	baseFeePerGas uint64                                            // Current base fee per gas on chain
	pendingTxs    cmapV2.ConcurrentMap[string, *PendingTransaction] // Mapping from nonce to pending transaction
	confirmCh     chan *Confirmation

	stopCh chan struct{}

	senderCheckBalancerTotal                *prometheus.CounterVec
	senderCheckPendingTransactionTotal      *prometheus.CounterVec
	sendTransactionTotal                    *prometheus.CounterVec
	sendTransactionFailureFullTx            *prometheus.GaugeVec
	sendTransactionFailureRepeatTransaction *prometheus.CounterVec
	sendTransactionFailureGetFee            *prometheus.CounterVec
	sendTransactionFailureSendTx            *prometheus.CounterVec
	resubmitTransactionTotal                *prometheus.CounterVec
	currentPendingTxsNum                    *prometheus.GaugeVec
	currentGasFeeCap                        *prometheus.GaugeVec
	currentGasTipCap                        *prometheus.GaugeVec
	currentGasPrice                         *prometheus.GaugeVec
	currentGasLimit                         *prometheus.GaugeVec
	currentNonce                            *prometheus.GaugeVec
}

// NewSender returns a new instance of transaction sender
// txConfirmationCh is used to notify confirmed transaction
func NewSender(ctx context.Context, config *config.SenderConfig, priv *ecdsa.PrivateKey, service, name string, reg prometheus.Registerer) (*Sender, error) {
	client, err := ethclient.Dial(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to dial eth client, err: %w", err)
	}

	// get chainID from client
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID, err: %w", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(priv, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor with chain ID %v, err: %w", chainID, err)
	}

	// Set pending nonce
	nonce, err := client.PendingNonceAt(ctx, auth.From)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending nonce for address %s, err: %w", auth.From.Hex(), err)
	}
	auth.Nonce = big.NewInt(int64(nonce))

	// get header by number
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get header by number, err: %w", err)
	}

	var baseFeePerGas uint64
	if config.TxType == DynamicFeeTxType {
		if header.BaseFee != nil {
			baseFeePerGas = header.BaseFee.Uint64()
		} else {
			return nil, errors.New("dynamic fee tx type not supported: header.BaseFee is nil")
		}
	}

	sender := &Sender{
		ctx:           ctx,
		config:        config,
		client:        client,
		chainID:       chainID,
		auth:          auth,
		minBalance:    config.MinBalance,
		confirmCh:     make(chan *Confirmation, 128),
		blockNumber:   header.Number.Uint64(),
		baseFeePerGas: baseFeePerGas,
		pendingTxs:    cmapV2.New[*PendingTransaction](),
		stopCh:        make(chan struct{}),
		name:          name,
		service:       service,

		sendTransactionTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "bridge_sender_send_transaction_total",
			Help: "The total number of sending transaction.",
		}, []string{"service", "name"}),
		sendTransactionFailureFullTx: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Name: "bridge_sender_send_transaction_full_tx_failure_total",
			Help: "The total number of sending transaction failure for full size tx.",
		}, []string{"service", "name"}),
		sendTransactionFailureRepeatTransaction: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "bridge_sender_send_transaction_repeat_transaction_failure_total",
			Help: "The total number of sending transaction failure for repeat transaction.",
		}, []string{"service", "name"}),
		sendTransactionFailureGetFee: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "bridge_sender_send_transaction_get_fee_failure_total",
			Help: "The total number of sending transaction failure for getting fee.",
		}, []string{"service", "name"}),
		sendTransactionFailureSendTx: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "bridge_sender_send_transaction_send_tx_failure_total",
			Help: "The total number of sending transaction failure for sending tx.",
		}, []string{"service", "name"}),
		resubmitTransactionTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "bridge_sender_send_transaction_resubmit_send_transaction_total",
			Help: "The total number of resubmit transaction.",
		}, []string{"service", "name"}),
		currentPendingTxsNum: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Name: "bridge_sender_pending_tx_count",
			Help: "The pending tx count in the sender.",
		}, []string{"service", "name"}),
		currentGasFeeCap: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Name: "bridge_sender_gas_fee_cap",
			Help: "The gas fee of current transaction.",
		}, []string{"service", "name"}),
		currentGasTipCap: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Name: "bridge_sender_gas_tip_cap",
			Help: "The gas tip of current transaction.",
		}, []string{"service", "name"}),
		currentGasPrice: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Name: "bridge_sender_gas_price_cap",
			Help: "The gas price of current transaction.",
		}, []string{"service", "name"}),
		currentGasLimit: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Name: "bridge_sender_gas_limit",
			Help: "The gas limit of current transaction.",
		}, []string{"service", "name"}),
		currentNonce: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Name: "bridge_sender_nonce",
			Help: "The nonce of current transaction.",
		}, []string{"service", "name"}),
		senderCheckPendingTransactionTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "bridge_sender_check_pending_transaction_total",
			Help: "The total number of check pending transaction.",
		}, []string{"service", "name"}),
		senderCheckBalancerTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "bridge_sender_check_balancer_total",
			Help: "The total number of check balancer.",
		}, []string{"service", "name"}),
	}

	go sender.loop(ctx)

	return sender, nil
}

// PendingCount returns the current number of pending txs.
func (s *Sender) PendingCount() int {
	return s.pendingTxs.Count()
}

// PendingLimit returns the maximum number of pending txs the sender can handle.
func (s *Sender) PendingLimit() int {
	return s.config.PendingLimit
}

// IsFull returns true if the sender's pending tx pool is full.
func (s *Sender) IsFull() bool {
	return s.pendingTxs.Count() >= s.config.PendingLimit
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

// SendConfirmation sends a confirmation to the confirmation channel.
// Note: This function is only used in tests.
func (s *Sender) SendConfirmation(cfm *Confirmation) {
	s.confirmCh <- cfm
}

func (s *Sender) getFeeData(auth *bind.TransactOpts, target *common.Address, value *big.Int, data []byte, minGasLimit uint64) (*FeeData, error) {
	if s.config.TxType == DynamicFeeTxType {
		return s.estimateDynamicGas(auth, target, value, data, minGasLimit)
	}
	return s.estimateLegacyGas(auth, target, value, data, minGasLimit)
}

// SendTransaction send a signed L2tL1 transaction.
func (s *Sender) SendTransaction(ID string, target *common.Address, value *big.Int, data []byte, minGasLimit uint64) (common.Hash, error) {
	s.sendTransactionTotal.WithLabelValues(s.service, s.name).Inc()
	if s.IsFull() {
		s.sendTransactionFailureFullTx.WithLabelValues(s.service, s.name).Set(1)
		return common.Hash{}, ErrFullPending
	}

	s.sendTransactionFailureFullTx.WithLabelValues(s.service, s.name).Set(0)
	if ok := s.pendingTxs.SetIfAbsent(ID, nil); !ok {
		s.sendTransactionFailureRepeatTransaction.WithLabelValues(s.service, s.name).Inc()
		return common.Hash{}, fmt.Errorf("repeat transaction ID: %s", ID)
	}

	var (
		feeData *FeeData
		tx      *types.Transaction
		err     error
	)

	defer func() {
		if err != nil {
			s.pendingTxs.Remove(ID) // release the ID on failure
		}
	}()

	if feeData, err = s.getFeeData(s.auth, target, value, data, minGasLimit); err != nil {
		s.sendTransactionFailureGetFee.WithLabelValues(s.service, s.name).Inc()
		return common.Hash{}, fmt.Errorf("failed to get fee data, err: %w", err)
	}

	if tx, err = s.createAndSendTx(s.auth, feeData, target, value, data, nil); err != nil {
		s.sendTransactionFailureSendTx.WithLabelValues(s.service, s.name).Inc()
		return common.Hash{}, fmt.Errorf("failed to create and send transaction, err: %w", err)
	}

	// add pending transaction
	pending := &PendingTransaction{
		tx:       tx,
		id:       ID,
		signer:   s.auth,
		submitAt: atomic.LoadUint64(&s.blockNumber),
		feeData:  feeData,
	}
	s.pendingTxs.Set(ID, pending)
	return tx.Hash(), nil
}

func (s *Sender) createAndSendTx(auth *bind.TransactOpts, feeData *FeeData, target *common.Address, value *big.Int, data []byte, overrideNonce *uint64) (*types.Transaction, error) {
	var (
		nonce  = auth.Nonce.Uint64()
		txData types.TxData
	)

	// this is a resubmit call, override the nonce
	if overrideNonce != nil {
		nonce = *overrideNonce
	}

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
	tx, err := auth.Signer(auth.From, types.NewTx(txData))
	if err != nil {
		log.Error("failed to sign tx", "err", err)
		return nil, err
	}
	if err = s.client.SendTransaction(s.ctx, tx); err != nil {
		log.Error("failed to send tx", "tx hash", tx.Hash().String(), "err", err)
		// Check if contain nonce, and reset nonce
		// only reset nonce when it is not from resubmit
		if strings.Contains(err.Error(), "nonce") && overrideNonce == nil {
			s.resetNonce(context.Background())
		}
		return nil, err
	}

	s.currentGasTipCap.WithLabelValues(s.service, s.name).Set(float64(feeData.gasTipCap.Int64()))
	s.currentGasFeeCap.WithLabelValues(s.service, s.name).Set(float64(feeData.gasFeeCap.Int64()))
	s.currentGasPrice.WithLabelValues(s.service, s.name).Set(float64(feeData.gasPrice.Int64()))
	s.currentGasLimit.WithLabelValues(s.service, s.name).Set(float64(feeData.gasLimit))
	s.currentNonce.WithLabelValues(s.service, s.name).Set(float64(auth.Nonce.Int64()))

	// update nonce when it is not from resubmit
	if overrideNonce == nil {
		auth.Nonce = big.NewInt(int64(nonce + 1))
	}
	return tx, nil
}

// reSetNonce reset nonce if send signed tx failed.
func (s *Sender) resetNonce(ctx context.Context) {
	nonce, err := s.client.PendingNonceAt(ctx, s.auth.From)
	if err != nil {
		log.Warn("failed to reset nonce", "address", s.auth.From.String(), "err", err)
		return
	}
	s.auth.Nonce = big.NewInt(int64(nonce))
}

func (s *Sender) resubmitTransaction(feeData *FeeData, auth *bind.TransactOpts, tx *types.Transaction) (*types.Transaction, error) {
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

		// adjust for rising basefee
		adjBaseFee := big.NewInt(0)
		if feeGas := atomic.LoadUint64(&s.baseFeePerGas); feeGas != 0 {
			adjBaseFee.SetUint64(feeGas)
		}
		adjBaseFee = adjBaseFee.Mul(adjBaseFee, escalateMultipleNum)
		adjBaseFee = adjBaseFee.Div(adjBaseFee, escalateMultipleDen)
		currentGasFeeCap := new(big.Int).Add(gasTipCap, adjBaseFee)
		if gasFeeCap.Cmp(currentGasFeeCap) < 0 {
			gasFeeCap = currentGasFeeCap
		}

		// but don't exceed maxGasPrice
		if gasFeeCap.Cmp(maxGasPrice) > 0 {
			gasFeeCap = maxGasPrice
		}
		feeData.gasFeeCap = gasFeeCap
		feeData.gasTipCap = gasTipCap
	}

	nonce := tx.Nonce()
	s.resubmitTransactionTotal.WithLabelValues(s.service, s.name).Inc()
	return s.createAndSendTx(auth, feeData, tx.To(), tx.Value(), tx.Data(), &nonce)
}

// checkPendingTransaction checks the confirmation status of pending transactions against the latest confirmed block number.
// If a transaction hasn't been confirmed after a certain number of blocks, it will be resubmitted with an increased gas price.
func (s *Sender) checkPendingTransaction(header *types.Header, confirmed uint64) {
	number := header.Number.Uint64()
	atomic.StoreUint64(&s.blockNumber, number)

	if s.config.TxType == DynamicFeeTxType {
		if header.BaseFee != nil {
			atomic.StoreUint64(&s.baseFeePerGas, header.BaseFee.Uint64())
		} else {
			log.Error("DynamicFeeTxType not supported, header.BaseFee nil")
		}
	}

	for item := range s.pendingTxs.IterBuffered() {
		key, pending := item.Key, item.Val
		// ignore empty id, since we use empty id to occupy pending task
		if pending == nil {
			continue
		}

		receipt, err := s.client.TransactionReceipt(s.ctx, pending.tx.Hash())
		if (err == nil) && (receipt != nil) {
			if receipt.BlockNumber.Uint64() <= confirmed {
				s.pendingTxs.Remove(key)
				// send confirm message
				s.confirmCh <- &Confirmation{
					ID:           pending.id,
					IsSuccessful: receipt.Status == types.ReceiptStatusSuccessful,
					TxHash:       pending.tx.Hash(),
				}
			}
		} else if s.config.EscalateBlocks+pending.submitAt < number {
			var tx *types.Transaction
			tx, err := s.resubmitTransaction(pending.feeData, pending.signer, pending.tx)
			if err != nil {
				// If account pool is empty, it will try again in next loop.
				if !errors.Is(err, ErrNoAvailableAccount) {
					log.Error("failed to resubmit transaction, reset submitAt", "tx hash", pending.tx.Hash().String(), "err", err)
				}
				// This means one of the old transactions is confirmed
				// One scenario is
				//   1. Initially, we replace the tx three times and submit it to local node.
				//      Currently, we keep the last tx hash in the memory.
				//   2. Other node packed the 2-nd tx or 3-rd tx, and the local node has received the block now.
				//   3. When we resubmit the 4-th tx, we got a nonce error.
				//   4. We need to check the status of 3-rd tx stored in our memory
				//     4.1 If the 3-rd tx is packed, we got a receipt and 3-nd is marked as confirmed.
				//     4.2 If the 2-nd tx is packed, we got nothing from `TransactionReceipt` call. Since we
				//         cannot do  anything about, we just log some information. In this case, the caller
				//         of `sender.SendTransaction` should write extra code to handle the situation.
				// Another scenario is private key leaking and someone send a transaction with the same nonce.
				// We need to stop the program and manually handle the situation.
				if strings.Contains(err.Error(), "nonce") {
					// This key can be deleted
					s.pendingTxs.Remove(key)
					// Try get receipt by the latest replaced tx hash
					receipt, err := s.client.TransactionReceipt(s.ctx, pending.tx.Hash())
					if (err == nil) && (receipt != nil) {
						// send confirm message
						s.confirmCh <- &Confirmation{
							ID:           pending.id,
							IsSuccessful: receipt.Status == types.ReceiptStatusSuccessful,
							TxHash:       pending.tx.Hash(),
						}
					} else {
						// The receipt can be nil since the confirmed transaction may not be the latest one.
						// We just ignore it, the caller of the sender pool should handle this situation.
						log.Warn("Pending transaction is confirmed by one of the replaced transactions", "key", key, "signer", pending.signer.From, "nonce", pending.tx.Nonce())
					}
				}
			} else {
				// flush submitAt
				pending.tx = tx
				pending.submitAt = number
			}
		}
	}
}

// checkBalance checks balance and print error log if balance is under a threshold.
func (s *Sender) checkBalance(ctx context.Context) error {
	bls, err := s.client.BalanceAt(ctx, s.auth.From, nil)
	if err != nil {
		log.Warn("failed to get balance", "address", s.auth.From.String(), "err", err)
		return err
	}

	if bls.Cmp(s.minBalance) < 0 {
		return fmt.Errorf("insufficient account balance - actual balance: %s, minimum required balance: %s",
			bls.String(), s.minBalance.String())
	}

	return nil
}

// Loop is the main event loop
func (s *Sender) loop(ctx context.Context) {
	checkTick := time.NewTicker(time.Duration(s.config.CheckPendingTime) * time.Second)
	defer checkTick.Stop()

	checkBalanceTicker := time.NewTicker(time.Duration(s.config.CheckBalanceTime) * time.Second)
	defer checkBalanceTicker.Stop()

	for {
		select {
		case <-checkTick.C:
			s.senderCheckPendingTransactionTotal.WithLabelValues(s.service, s.name).Inc()
			header, err := s.client.HeaderByNumber(s.ctx, nil)
			if err != nil {
				log.Error("failed to get latest head", "err", err)
				continue
			}

			confirmed, err := utils.GetLatestConfirmedBlockNumber(s.ctx, s.client, s.config.Confirmations)
			if err != nil {
				log.Error("failed to get latest confirmed block number", "err", err)
				continue
			}

			s.checkPendingTransaction(header, confirmed)
		case <-checkBalanceTicker.C:
			s.senderCheckBalancerTotal.WithLabelValues(s.service, s.name).Inc()
			// Check and set balance.
			if err := s.checkBalance(ctx); err != nil {
				log.Error("check balance error", "err", err)
			}
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		}
	}
}
