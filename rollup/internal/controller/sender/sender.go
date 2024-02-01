package sender

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/ethclient/gethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rlp"
	"github.com/scroll-tech/go-ethereum/rpc"
	"gorm.io/gorm"

	"scroll-tech/common/types"

	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/orm"
	"scroll-tech/rollup/internal/utils"
)

const (
	// AccessListTxType type for AccessListTx
	AccessListTxType = "AccessListTx"

	// DynamicFeeTxType type for DynamicFeeTx
	DynamicFeeTxType = "DynamicFeeTx"

	// LegacyTxType type for LegacyTx
	LegacyTxType = "LegacyTx"
)

// Confirmation struct used to indicate transaction confirmation details
type Confirmation struct {
	ContextID    string
	IsSuccessful bool
	TxHash       common.Hash
	SenderType   types.SenderType
}

// FeeData fee struct used to estimate gas price
type FeeData struct {
	gasFeeCap *big.Int
	gasTipCap *big.Int
	gasPrice  *big.Int

	accessList gethTypes.AccessList

	gasLimit uint64
}

// Sender Transaction sender to send transaction to l1/l2 geth
type Sender struct {
	config     *config.SenderConfig
	gethClient *gethclient.Client
	client     *ethclient.Client // The client to retrieve on chain data or send transaction.
	chainID    *big.Int          // The chain id of the endpoint
	ctx        context.Context
	service    string
	name       string
	senderType types.SenderType

	auth *bind.TransactOpts

	db                    *gorm.DB
	pendingTransactionOrm *orm.PendingTransaction

	confirmCh chan *Confirmation
	stopCh    chan struct{}

	metrics *senderMetrics
}

// NewSender returns a new instance of transaction sender
func NewSender(ctx context.Context, config *config.SenderConfig, priv *ecdsa.PrivateKey, service, name string, senderType types.SenderType, db *gorm.DB, reg prometheus.Registerer) (*Sender, error) {
	if config.EscalateMultipleNum <= config.EscalateMultipleDen {
		return nil, fmt.Errorf("invalid params, EscalateMultipleNum; %v, EscalateMultipleDen: %v", config.EscalateMultipleNum, config.EscalateMultipleDen)
	}

	rpcClient, err := rpc.Dial(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to dial eth client, err: %w", err)
	}

	client := ethclient.NewClient(rpcClient)
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

	sender := &Sender{
		ctx:                   ctx,
		config:                config,
		gethClient:            gethclient.New(rpcClient),
		client:                client,
		chainID:               chainID,
		auth:                  auth,
		db:                    db,
		pendingTransactionOrm: orm.NewPendingTransaction(db),
		confirmCh:             make(chan *Confirmation, 128),
		stopCh:                make(chan struct{}),
		name:                  name,
		service:               service,
		senderType:            senderType,
	}
	sender.metrics = initSenderMetrics(reg)

	go sender.loop(ctx)

	return sender, nil
}

// GetChainID returns the chain ID associated with the sender.
func (s *Sender) GetChainID() *big.Int {
	return s.chainID
}

// Stop stop the sender module.
func (s *Sender) Stop() {
	close(s.stopCh)
	log.Info("sender stopped", "name", s.name, "service", s.service, "address", s.auth.From.String())
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

func (s *Sender) getFeeData(target *common.Address, value *big.Int, data []byte, fallbackGasLimit uint64, baseFee uint64) (*FeeData, error) {
	if s.config.TxType == DynamicFeeTxType {
		return s.estimateDynamicGas(target, value, data, fallbackGasLimit, baseFee)
	}
	return s.estimateLegacyGas(target, value, data, fallbackGasLimit)
}

// SendTransaction send a signed L2tL1 transaction.
func (s *Sender) SendTransaction(contextID string, target *common.Address, value *big.Int, data []byte, fallbackGasLimit uint64) (common.Hash, error) {
	s.metrics.sendTransactionTotal.WithLabelValues(s.service, s.name).Inc()
	var (
		feeData *FeeData
		tx      *gethTypes.Transaction
		err     error
	)

	blockNumber, baseFee, err := s.getBlockNumberAndBaseFee(s.ctx)
	if err != nil {
		log.Error("failed to get block number and base fee", "error", err)
		return common.Hash{}, fmt.Errorf("failed to get block number and base fee, err: %w", err)
	}

	if feeData, err = s.getFeeData(target, value, data, fallbackGasLimit, baseFee); err != nil {
		s.metrics.sendTransactionFailureGetFee.WithLabelValues(s.service, s.name).Inc()
		log.Error("failed to get fee data", "from", s.auth.From.String(), "nonce", s.auth.Nonce.Uint64(), "fallback gas limit", fallbackGasLimit, "err", err)
		return common.Hash{}, fmt.Errorf("failed to get fee data, err: %w", err)
	}

	if tx, err = s.createAndSendTx(feeData, target, value, data, nil); err != nil {
		s.metrics.sendTransactionFailureSendTx.WithLabelValues(s.service, s.name).Inc()
		log.Error("failed to create and send tx (non-resubmit case)", "from", s.auth.From.String(), "nonce", s.auth.Nonce.Uint64(), "err", err)
		return common.Hash{}, fmt.Errorf("failed to create and send transaction, err: %w", err)
	}

	if err = s.pendingTransactionOrm.InsertPendingTransaction(s.ctx, contextID, s.getSenderMeta(), tx, blockNumber); err != nil {
		log.Error("failed to insert transaction", "from", s.auth.From.String(), "nonce", s.auth.Nonce.Uint64(), "err", err)
		return common.Hash{}, fmt.Errorf("failed to insert transaction, err: %w", err)
	}
	return tx.Hash(), nil
}

func (s *Sender) createAndSendTx(feeData *FeeData, target *common.Address, value *big.Int, data []byte, overrideNonce *uint64) (*gethTypes.Transaction, error) {
	var (
		nonce  = s.auth.Nonce.Uint64()
		txData gethTypes.TxData
	)

	// this is a resubmit call, override the nonce
	if overrideNonce != nil {
		nonce = *overrideNonce
	}

	switch s.config.TxType {
	case LegacyTxType:
		// for ganache mock node
		txData = &gethTypes.LegacyTx{
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
		txData = &gethTypes.AccessListTx{
			ChainID:    s.chainID,
			Nonce:      nonce,
			GasPrice:   feeData.gasPrice,
			Gas:        feeData.gasLimit,
			To:         target,
			Value:      new(big.Int).Set(value),
			Data:       common.CopyBytes(data),
			AccessList: feeData.accessList,
			V:          new(big.Int),
			R:          new(big.Int),
			S:          new(big.Int),
		}
	default:
		txData = &gethTypes.DynamicFeeTx{
			Nonce:      nonce,
			To:         target,
			Data:       common.CopyBytes(data),
			Gas:        feeData.gasLimit,
			AccessList: feeData.accessList,
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
	tx, err := s.auth.Signer(s.auth.From, gethTypes.NewTx(txData))
	if err != nil {
		log.Error("failed to sign tx", "address", s.auth.From.String(), "err", err)
		return nil, err
	}

	if err = s.client.SendTransaction(s.ctx, tx); err != nil {
		log.Error("failed to send tx", "tx hash", tx.Hash().String(), "from", s.auth.From.String(), "nonce", tx.Nonce(), "err", err)
		// Check if contain nonce, and reset nonce
		// only reset nonce when it is not from resubmit
		if strings.Contains(err.Error(), "nonce") && overrideNonce == nil {
			s.resetNonce(context.Background())
		}
		return nil, err
	}

	if feeData.gasTipCap != nil {
		s.metrics.currentGasTipCap.WithLabelValues(s.service, s.name).Set(float64(feeData.gasTipCap.Uint64()))
	}

	if feeData.gasFeeCap != nil {
		s.metrics.currentGasFeeCap.WithLabelValues(s.service, s.name).Set(float64(feeData.gasFeeCap.Uint64()))
	}

	if feeData.gasPrice != nil {
		s.metrics.currentGasPrice.WithLabelValues(s.service, s.name).Set(float64(feeData.gasPrice.Uint64()))
	}

	s.metrics.currentGasLimit.WithLabelValues(s.service, s.name).Set(float64(feeData.gasLimit))

	// update nonce when it is not from resubmit
	if overrideNonce == nil {
		s.auth.Nonce = big.NewInt(int64(nonce + 1))
	}
	return tx, nil
}

// resetNonce reset nonce if send signed tx failed.
func (s *Sender) resetNonce(ctx context.Context) {
	nonce, err := s.client.PendingNonceAt(ctx, s.auth.From)
	if err != nil {
		log.Warn("failed to reset nonce", "address", s.auth.From.String(), "err", err)
		return
	}
	s.auth.Nonce = big.NewInt(int64(nonce))
}

func (s *Sender) resubmitTransaction(tx *gethTypes.Transaction, baseFee uint64) (*gethTypes.Transaction, error) {
	escalateMultipleNum := new(big.Int).SetUint64(s.config.EscalateMultipleNum)
	escalateMultipleDen := new(big.Int).SetUint64(s.config.EscalateMultipleDen)
	maxGasPrice := new(big.Int).SetUint64(s.config.MaxGasPrice)

	txInfo := map[string]interface{}{
		"tx_hash": tx.Hash().String(),
		"tx_type": s.config.TxType,
		"from":    s.auth.From.String(),
		"nonce":   tx.Nonce(),
	}

	var feeData FeeData
	feeData.gasLimit = tx.Gas()
	switch s.config.TxType {
	case LegacyTxType, AccessListTxType: // `LegacyTxType`is for ganache mock node
		originalGasPrice := tx.GasPrice()
		gasPrice := new(big.Int).Mul(escalateMultipleNum, originalGasPrice)
		gasPrice = gasPrice.Div(gasPrice, escalateMultipleDen)
		if gasPrice.Cmp(maxGasPrice) > 0 {
			gasPrice = maxGasPrice
		}

		if originalGasPrice.Cmp(gasPrice) == 0 {
			log.Warn("gas price bump corner case, add 1 wei", "original", originalGasPrice.Uint64(), "adjusted", gasPrice.Uint64())
			gasPrice = new(big.Int).Add(gasPrice, big.NewInt(1))
		}

		feeData.gasPrice = gasPrice
		txInfo["original_gas_price"] = originalGasPrice.Uint64()
		txInfo["adjusted_gas_price"] = gasPrice.Uint64()
	default:
		originalGasTipCap := tx.GasTipCap()
		originalGasFeeCap := tx.GasFeeCap()

		gasTipCap := new(big.Int).Mul(originalGasTipCap, escalateMultipleNum)
		gasTipCap = gasTipCap.Div(gasTipCap, escalateMultipleDen)
		gasFeeCap := new(big.Int).Mul(originalGasFeeCap, escalateMultipleNum)
		gasFeeCap = gasFeeCap.Div(gasFeeCap, escalateMultipleDen)

		// adjust for rising basefee
		adjBaseFee := new(big.Int).SetUint64(baseFee)
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

		// gasTipCap <= gasFeeCap
		if gasTipCap.Cmp(gasFeeCap) > 0 {
			gasTipCap = gasFeeCap
		}

		if originalGasTipCap.Cmp(gasTipCap) == 0 {
			log.Warn("gas tip cap bump corner case, add 1 wei", "original", originalGasTipCap.Uint64(), "adjusted", gasTipCap.Uint64())
			gasTipCap = new(big.Int).Add(gasTipCap, big.NewInt(1))
		}

		if originalGasFeeCap.Cmp(gasFeeCap) == 0 {
			log.Warn("gas fee cap bump corner case, add 1 wei", "original", originalGasFeeCap.Uint64(), "adjusted", gasFeeCap.Uint64())
			gasFeeCap = new(big.Int).Add(gasFeeCap, big.NewInt(1))
		}

		feeData.gasFeeCap = gasFeeCap
		feeData.gasTipCap = gasTipCap
		txInfo["original_gas_tip_cap"] = originalGasTipCap.Uint64()
		txInfo["adjusted_gas_tip_cap"] = gasTipCap.Uint64()
		txInfo["original_gas_fee_cap"] = originalGasFeeCap.Uint64()
		txInfo["adjusted_gas_fee_cap"] = gasFeeCap.Uint64()
	}

	log.Info("Transaction gas adjustment details", "service", s.service, "name", s.name, "txInfo", txInfo)

	nonce := tx.Nonce()
	s.metrics.resubmitTransactionTotal.WithLabelValues(s.service, s.name).Inc()
	tx, err := s.createAndSendTx(&feeData, tx.To(), tx.Value(), tx.Data(), &nonce)
	if err != nil {
		log.Error("failed to create and send tx (resubmit case)", "from", s.auth.From.String(), "nonce", nonce, "err", err)
		return nil, err
	}
	return tx, nil
}

// checkPendingTransaction checks the confirmation status of pending transactions against the latest confirmed block number.
// If a transaction hasn't been confirmed after a certain number of blocks, it will be resubmitted with an increased gas price.
func (s *Sender) checkPendingTransaction() {
	s.metrics.senderCheckPendingTransactionTotal.WithLabelValues(s.service, s.name).Inc()

	blockNumber, baseFee, err := s.getBlockNumberAndBaseFee(s.ctx)
	if err != nil {
		log.Error("failed to get block number and base fee", "error", err)
		return
	}

	transactionsToCheck, err := s.pendingTransactionOrm.GetPendingOrReplacedTransactionsBySenderType(s.ctx, s.senderType, 100)
	if err != nil {
		log.Error("failed to load pending transactions", "sender meta", s.getSenderMeta(), "err", err)
		return
	}

	confirmed, err := utils.GetLatestConfirmedBlockNumber(s.ctx, s.client, s.config.Confirmations)
	if err != nil {
		log.Error("failed to get latest confirmed block number", "confirmations", s.config.Confirmations, "err", err)
		return
	}

	for _, txnToCheck := range transactionsToCheck {
		tx := new(gethTypes.Transaction)
		if err := tx.DecodeRLP(rlp.NewStream(bytes.NewReader(txnToCheck.RLPEncoding), 0)); err != nil {
			log.Error("failed to decode RLP", "context ID", txnToCheck.ContextID, "sender meta", s.getSenderMeta(), "err", err)
			continue
		}

		receipt, err := s.client.TransactionReceipt(s.ctx, tx.Hash())
		if (err == nil) && (receipt != nil) { // tx confirmed.
			if receipt.BlockNumber.Uint64() <= confirmed {
				err := s.db.Transaction(func(dbTX *gorm.DB) error {
					// Update the status of the transaction to TxStatusConfirmed.
					if err := s.pendingTransactionOrm.UpdatePendingTransactionStatusByTxHash(s.ctx, tx.Hash(), types.TxStatusConfirmed, dbTX); err != nil {
						log.Error("failed to update transaction status by tx hash", "hash", tx.Hash().String(), "sender meta", s.getSenderMeta(), "from", s.auth.From.String(), "nonce", tx.Nonce(), "err", err)
						return err
					}
					// Update other transactions with the same nonce and sender address as failed.
					if err := s.pendingTransactionOrm.UpdateOtherTransactionsAsFailedByNonce(s.ctx, txnToCheck.SenderAddress, tx.Nonce(), tx.Hash(), dbTX); err != nil {
						log.Error("failed to update other transactions as failed by nonce", "senderAddress", txnToCheck.SenderAddress, "nonce", tx.Nonce(), "excludedTxHash", tx.Hash(), "err", err)
						return err
					}
					return nil
				})
				if err != nil {
					log.Error("db transaction failed after receiving confirmation", "err", err)
					return
				}

				// send confirm message
				s.confirmCh <- &Confirmation{
					ContextID:    txnToCheck.ContextID,
					IsSuccessful: receipt.Status == gethTypes.ReceiptStatusSuccessful,
					TxHash:       tx.Hash(),
					SenderType:   s.senderType,
				}
			}
		} else if txnToCheck.Status == types.TxStatusPending && // Only try resubmitting a new transaction based on gas price of the last transaction (status pending) with same ContextID.
			s.config.EscalateBlocks+txnToCheck.SubmitBlockNumber <= blockNumber {
			// It's possible that the pending transaction was marked as failed earlier in this loop (e.g., if one of its replacements has already been confirmed).
			// Therefore, we fetch the current transaction status again for accuracy before proceeding.
			status, err := s.pendingTransactionOrm.GetTxStatusByTxHash(s.ctx, tx.Hash())
			if err != nil {
				log.Error("failed to get transaction status by tx hash", "hash", tx.Hash().String(), "err", err)
				return
			}
			if status == types.TxStatusConfirmedFailed {
				log.Warn("transaction already marked as failed, skipping resubmission", "hash", tx.Hash().String())
				continue
			}

			log.Info("resubmit transaction",
				"service", s.service,
				"name", s.name,
				"hash", tx.Hash().String(),
				"from", s.auth.From.String(),
				"nonce", tx.Nonce(),
				"submitBlockNumber", txnToCheck.SubmitBlockNumber,
				"currentBlockNumber", blockNumber,
				"escalateBlocks", s.config.EscalateBlocks)

			if newTx, err := s.resubmitTransaction(tx, baseFee); err != nil {
				s.metrics.resubmitTransactionFailedTotal.WithLabelValues(s.service, s.name).Inc()
				log.Error("failed to resubmit transaction", "context ID", txnToCheck.ContextID, "sender meta", s.getSenderMeta(), "from", s.auth.From.String(), "nonce", newTx.Nonce(), "err", err)
			} else {
				err := s.db.Transaction(func(dbTX *gorm.DB) error {
					// Update the status of the original transaction as replaced, while still checking its confirmation status.
					if err := s.pendingTransactionOrm.UpdatePendingTransactionStatusByTxHash(s.ctx, tx.Hash(), types.TxStatusReplaced, dbTX); err != nil {
						return fmt.Errorf("failed to update status of transaction with hash %s to TxStatusReplaced, err: %w", tx.Hash().String(), err)
					}
					// Record the new transaction that has replaced the original one.
					if err := s.pendingTransactionOrm.InsertPendingTransaction(s.ctx, txnToCheck.ContextID, s.getSenderMeta(), newTx, blockNumber, dbTX); err != nil {
						return fmt.Errorf("failed to insert new pending transaction with context ID: %s, nonce: %d, hash: %v, previous block number: %v, current block number: %v, err: %w", txnToCheck.ContextID, newTx.Nonce(), newTx.Hash().String(), txnToCheck.SubmitBlockNumber, blockNumber, err)
					}
					return nil
				})
				if err != nil {
					log.Error("db transaction failed after resubmitting", "err", err)
					return
				}
			}
		}
	}
}

// Loop is the main event loop
func (s *Sender) loop(ctx context.Context) {
	checkTick := time.NewTicker(time.Duration(s.config.CheckPendingTime) * time.Second)
	defer checkTick.Stop()

	for {
		select {
		case <-checkTick.C:
			s.checkPendingTransaction()
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		}
	}
}

func (s *Sender) getSenderMeta() *orm.SenderMeta {
	return &orm.SenderMeta{
		Name:    s.name,
		Service: s.service,
		Address: s.auth.From,
		Type:    s.senderType,
	}
}

func (s *Sender) getBlockNumberAndBaseFee(ctx context.Context) (uint64, uint64, error) {
	header, err := s.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get header by number, err: %w", err)
	}

	var baseFeePerGas uint64
	if s.config.TxType == DynamicFeeTxType {
		if header.BaseFee != nil {
			baseFeePerGas = header.BaseFee.Uint64()
		} else {
			return 0, 0, errors.New("dynamic fee tx type not supported: header.BaseFee is nil")
		}
	}
	return header.Number.Uint64(), baseFeePerGas, nil
}
