package sender

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/holiman/uint256"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/consensus/misc/eip4844"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
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
	// LegacyTxType type for LegacyTx
	LegacyTxType = "LegacyTx"

	// DynamicFeeTxType type for DynamicFeeTx
	DynamicFeeTxType = "DynamicFeeTx"
)

var (
	// ErrTooManyPendingBlobTxs error for too many pending blob txs
	ErrTooManyPendingBlobTxs = errors.New("the limit of pending blob-carrying transactions has been exceeded")
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

	blobGasFeeCap *big.Int

	accessList gethTypes.AccessList

	gasLimit uint64
}

// Sender Transaction sender to send transaction to l1/l2 geth
type Sender struct {
	config            *config.SenderConfig
	gethClient        *gethclient.Client
	client            *ethclient.Client // The client to retrieve on chain data or send transaction.
	transactionSigner *TransactionSigner
	chainID           *big.Int // The chain id of the endpoint
	ctx               context.Context
	service           string
	name              string
	senderType        types.SenderType

	db                    *gorm.DB
	pendingTransactionOrm *orm.PendingTransaction

	confirmCh chan *Confirmation
	stopCh    chan struct{}

	metrics *senderMetrics
}

// NewSender returns a new instance of transaction sender
func NewSender(ctx context.Context, config *config.SenderConfig, signerConfig *config.SignerConfig, service, name string, senderType types.SenderType, db *gorm.DB, reg prometheus.Registerer) (*Sender, error) {
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
	transactionSigner, err := NewTransactionSigner(signerConfig, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction signer, err: %w", err)
	}

	// Set pending nonce
	nonce, err := client.PendingNonceAt(ctx, transactionSigner.GetAddr())
	if err != nil {
		return nil, fmt.Errorf("failed to get pending nonce for address %s, err: %w", transactionSigner.GetAddr(), err)
	}
	transactionSigner.SetNonce(nonce)

	sender := &Sender{
		ctx:                   ctx,
		config:                config,
		gethClient:            gethclient.New(rpcClient),
		client:                client,
		chainID:               chainID,
		transactionSigner:     transactionSigner,
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
	log.Info("sender stopped", "name", s.name, "service", s.service, "address", s.transactionSigner.GetAddr().String())
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

func (s *Sender) getFeeData(target *common.Address, data []byte, sidecar *gethTypes.BlobTxSidecar, baseFee, blobBaseFee uint64, fallbackGasLimit uint64) (*FeeData, error) {
	switch s.config.TxType {
	case LegacyTxType:
		return s.estimateLegacyGas(target, data, fallbackGasLimit)
	case DynamicFeeTxType:
		if sidecar == nil {
			return s.estimateDynamicGas(target, data, baseFee, fallbackGasLimit)
		}
		return s.estimateBlobGas(target, data, sidecar, baseFee, blobBaseFee, fallbackGasLimit)
	default:
		return nil, fmt.Errorf("unsupported transaction type: %s", s.config.TxType)
	}
}

// SendTransaction send a signed L2tL1 transaction.
func (s *Sender) SendTransaction(contextID string, target *common.Address, data []byte, blob *kzg4844.Blob, fallbackGasLimit uint64) (common.Hash, error) {
	s.metrics.sendTransactionTotal.WithLabelValues(s.service, s.name).Inc()
	var (
		feeData *FeeData
		sidecar *gethTypes.BlobTxSidecar
		err     error
	)

	if blob != nil {
		// check that number of pending blob-carrying txs is not too big
		if s.senderType == types.SenderTypeCommitBatch {
			var numPendingTransactions int64
			// We should count here only blob-carrying txs, but due to check that blob != nil, we know that we already switched to blobs.
			// Now all txs with SenderTypeCommitBatch will be blob-carrying, but some of previous pending txs could still be non-blob.
			// But this can happen only once at the moment of switching from non-blob to blob (pre-Bernoulli and post-Bernoulli) and it doesn't break anything.
			// So don't need to add check that tx carries blob
			numPendingTransactions, err = s.pendingTransactionOrm.GetCountPendingTransactionsBySenderType(s.ctx, s.senderType)
			if err != nil {
				log.Error("failed to count pending transactions", "err: %w", err)
				return common.Hash{}, fmt.Errorf("failed to count pending transactions, err: %w", err)
			}
			if numPendingTransactions >= s.config.MaxPendingBlobTxs {
				return common.Hash{}, ErrTooManyPendingBlobTxs
			}

		}
		sidecar, err = makeSidecar(blob)
		if err != nil {
			log.Error("failed to make sidecar for blob transaction", "error", err)
			return common.Hash{}, fmt.Errorf("failed to make sidecar for blob transaction, err: %w", err)
		}
	}

	blockNumber, baseFee, blobBaseFee, err := s.getBlockNumberAndBaseFeeAndBlobFee(s.ctx)
	if err != nil {
		log.Error("failed to get block number and base fee", "error", err)
		return common.Hash{}, fmt.Errorf("failed to get block number and base fee, err: %w", err)
	}

	if feeData, err = s.getFeeData(target, data, sidecar, baseFee, blobBaseFee, fallbackGasLimit); err != nil {
		s.metrics.sendTransactionFailureGetFee.WithLabelValues(s.service, s.name).Inc()
		log.Error("failed to get fee data", "from", s.transactionSigner.GetAddr().String(), "nonce", s.transactionSigner.GetNonce(), "fallback gas limit", fallbackGasLimit, "err", err)
		return common.Hash{}, fmt.Errorf("failed to get fee data, err: %w", err)
	}

	signedTx, err := s.createTx(feeData, target, data, sidecar, s.transactionSigner.GetNonce())
	if err != nil {
		s.metrics.sendTransactionFailureSendTx.WithLabelValues(s.service, s.name).Inc()
		log.Error("failed to create signed tx (non-resubmit case)", "from", s.transactionSigner.GetAddr().String(), "nonce", s.transactionSigner.GetNonce(), "err", err)
		return common.Hash{}, fmt.Errorf("failed to create signed transaction, err: %w", err)
	}

	// Insert the transaction into the pending transaction table.
	// A corner case is that the transaction is inserted into the table but not sent to the chain, because the server is stopped in the middle.
	// This case will be handled by the checkPendingTransaction function.
	if err = s.pendingTransactionOrm.InsertPendingTransaction(s.ctx, contextID, s.getSenderMeta(), signedTx, blockNumber); err != nil {
		log.Error("failed to insert transaction", "from", s.transactionSigner.GetAddr().String(), "nonce", s.transactionSigner.GetNonce(), "err", err)
		return common.Hash{}, fmt.Errorf("failed to insert transaction, err: %w", err)
	}

	if err := s.client.SendTransaction(s.ctx, signedTx); err != nil {
		// Delete the transaction from the pending transaction table if it fails to send.
		if updateErr := s.pendingTransactionOrm.DeleteTransactionByTxHash(s.ctx, signedTx.Hash()); updateErr != nil {
			log.Error("failed to delete transaction", "tx hash", signedTx.Hash().String(), "from", s.transactionSigner.GetAddr().String(), "nonce", signedTx.Nonce(), "err", updateErr)
			return common.Hash{}, fmt.Errorf("failed to delete transaction, err: %w", updateErr)
		}

		log.Error("failed to send tx", "tx hash", signedTx.Hash().String(), "from", s.transactionSigner.GetAddr().String(), "nonce", signedTx.Nonce(), "err", err)
		// Check if contain nonce, and reset nonce
		// only reset nonce when it is not from resubmit
		if strings.Contains(err.Error(), "nonce too low") {
			s.resetNonce(context.Background())
		}
		return common.Hash{}, fmt.Errorf("failed to send transaction, err: %w", err)
	}

	s.transactionSigner.SetNonce(signedTx.Nonce() + 1)

	return signedTx.Hash(), nil
}

func (s *Sender) createTx(feeData *FeeData, target *common.Address, data []byte, sidecar *gethTypes.BlobTxSidecar, nonce uint64) (*gethTypes.Transaction, error) {
	var txData gethTypes.TxData

	switch s.config.TxType {
	case LegacyTxType:
		txData = &gethTypes.LegacyTx{
			Nonce:    nonce,
			GasPrice: feeData.gasPrice,
			Gas:      feeData.gasLimit,
			To:       target,
			Data:     data,
		}
	case DynamicFeeTxType:
		if sidecar == nil {
			txData = &gethTypes.DynamicFeeTx{
				Nonce:      nonce,
				To:         target,
				Data:       data,
				Gas:        feeData.gasLimit,
				AccessList: feeData.accessList,
				ChainID:    s.chainID,
				GasTipCap:  feeData.gasTipCap,
				GasFeeCap:  feeData.gasFeeCap,
			}
		} else {
			if target == nil {
				log.Error("blob transaction to address cannot be nil", "address", s.transactionSigner.GetAddr().String(), "chainID", s.chainID.Uint64(), "nonce", s.transactionSigner.GetNonce())
				return nil, errors.New("blob transaction to address cannot be nil")
			}

			txData = &gethTypes.BlobTx{
				ChainID:    uint256.MustFromBig(s.chainID),
				Nonce:      nonce,
				GasTipCap:  uint256.MustFromBig(feeData.gasTipCap),
				GasFeeCap:  uint256.MustFromBig(feeData.gasFeeCap),
				Gas:        feeData.gasLimit,
				To:         *target,
				Data:       data,
				AccessList: feeData.accessList,
				BlobFeeCap: uint256.MustFromBig(feeData.blobGasFeeCap),
				BlobHashes: sidecar.BlobHashes(),
				Sidecar:    sidecar,
			}
		}
	}

	// sign and send
	tx := gethTypes.NewTx(txData)
	signedTx, err := s.transactionSigner.SignTransaction(s.ctx, tx)
	if err != nil {
		log.Error("failed to sign tx", "address", s.transactionSigner.GetAddr().String(), "err", err)
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

	if feeData.blobGasFeeCap != nil {
		s.metrics.currentBlobGasFeeCap.WithLabelValues(s.service, s.name).Set(float64(feeData.blobGasFeeCap.Uint64()))
	}

	s.metrics.currentGasLimit.WithLabelValues(s.service, s.name).Set(float64(feeData.gasLimit))

	return signedTx, nil
}

// resetNonce reset nonce if send signed tx failed.
func (s *Sender) resetNonce(ctx context.Context) {
	nonce, err := s.client.PendingNonceAt(ctx, s.transactionSigner.GetAddr())
	if err != nil {
		log.Warn("failed to reset nonce", "address", s.transactionSigner.GetAddr().String(), "err", err)
		return
	}
	s.transactionSigner.SetNonce(nonce)
}

func (s *Sender) createReplacingTransaction(tx *gethTypes.Transaction, baseFee, blobBaseFee uint64) (*gethTypes.Transaction, error) {
	escalateMultipleNum := new(big.Int).SetUint64(s.config.EscalateMultipleNum)
	escalateMultipleDen := new(big.Int).SetUint64(s.config.EscalateMultipleDen)
	maxGasPrice := new(big.Int).SetUint64(s.config.MaxGasPrice)
	maxBlobGasPrice := new(big.Int).SetUint64(s.config.MaxBlobGasPrice)

	txInfo := map[string]interface{}{
		"tx_hash": tx.Hash().String(),
		"tx_type": s.config.TxType,
		"from":    s.transactionSigner.GetAddr().String(),
		"nonce":   tx.Nonce(),
	}

	var feeData FeeData
	feeData.gasLimit = tx.Gas()
	switch s.config.TxType {
	case LegacyTxType:
		originalGasPrice := tx.GasPrice()
		gasPrice := new(big.Int).Mul(originalGasPrice, escalateMultipleNum)
		gasPrice = new(big.Int).Div(gasPrice, escalateMultipleDen)
		baseFeeInt := new(big.Int).SetUint64(baseFee)
		if gasPrice.Cmp(baseFeeInt) < 0 {
			gasPrice = baseFeeInt
		}
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

	case DynamicFeeTxType:
		if tx.BlobTxSidecar() == nil {
			originalGasTipCap := tx.GasTipCap()
			originalGasFeeCap := tx.GasFeeCap()

			gasTipCap := new(big.Int).Mul(originalGasTipCap, escalateMultipleNum)
			gasTipCap = new(big.Int).Div(gasTipCap, escalateMultipleDen)
			gasFeeCap := new(big.Int).Mul(originalGasFeeCap, escalateMultipleNum)
			gasFeeCap = new(big.Int).Div(gasFeeCap, escalateMultipleDen)

			// adjust for rising basefee
			currentGasFeeCap := getGasFeeCap(new(big.Int).SetUint64(baseFee), gasTipCap)
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
		} else {
			originalGasTipCap := tx.GasTipCap()
			originalGasFeeCap := tx.GasFeeCap()
			originalBlobGasFeeCap := tx.BlobGasFeeCap()

			// bumping at least 100%
			gasTipCap := new(big.Int).Mul(originalGasTipCap, big.NewInt(2))
			gasFeeCap := new(big.Int).Mul(originalGasFeeCap, big.NewInt(2))
			blobGasFeeCap := new(big.Int).Mul(originalBlobGasFeeCap, big.NewInt(2))

			// adjust for rising basefee
			currentGasFeeCap := getGasFeeCap(new(big.Int).SetUint64(baseFee), gasTipCap)
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

			// adjust for rising blobbasefee
			currentBlobGasFeeCap := getBlobGasFeeCap(new(big.Int).SetUint64(blobBaseFee))
			if blobGasFeeCap.Cmp(currentBlobGasFeeCap) < 0 {
				blobGasFeeCap = currentBlobGasFeeCap
			}

			// but don't exceed maxBlobGasPrice
			if blobGasFeeCap.Cmp(maxBlobGasPrice) > 0 {
				blobGasFeeCap = maxBlobGasPrice
			}

			// Check if any fee cap is less than double
			doubledTipCap := new(big.Int).Mul(originalGasTipCap, big.NewInt(2))
			doubledFeeCap := new(big.Int).Mul(originalGasFeeCap, big.NewInt(2))
			doubledBlobFeeCap := new(big.Int).Mul(originalBlobGasFeeCap, big.NewInt(2))
			if gasTipCap.Cmp(doubledTipCap) < 0 || gasFeeCap.Cmp(doubledFeeCap) < 0 || blobGasFeeCap.Cmp(doubledBlobFeeCap) < 0 {
				log.Error("gas fees must be at least double", "originalTipCap", originalGasTipCap, "currentTipCap", gasTipCap, "requiredTipCap", doubledTipCap, "originalFeeCap", originalGasFeeCap, "currentFeeCap", gasFeeCap, "requiredFeeCap", doubledFeeCap, "originalBlobFeeCap", originalBlobGasFeeCap, "currentBlobFeeCap", blobGasFeeCap, "requiredBlobFeeCap", doubledBlobFeeCap)
				return nil, errors.New("gas fees must be at least double")
			}

			feeData.gasFeeCap = gasFeeCap
			feeData.gasTipCap = gasTipCap
			feeData.blobGasFeeCap = blobGasFeeCap
			txInfo["original_gas_tip_cap"] = originalGasTipCap.Uint64()
			txInfo["adjusted_gas_tip_cap"] = gasTipCap.Uint64()
			txInfo["original_gas_fee_cap"] = originalGasFeeCap.Uint64()
			txInfo["adjusted_gas_fee_cap"] = gasFeeCap.Uint64()
			txInfo["original_blob_gas_fee_cap"] = originalBlobGasFeeCap.Uint64()
			txInfo["adjusted_blob_gas_fee_cap"] = blobGasFeeCap.Uint64()
		}

	default:
		return nil, fmt.Errorf("unsupported transaction type: %s", s.config.TxType)
	}

	log.Info("Transaction gas adjustment details", "service", s.service, "name", s.name, "txInfo", txInfo)

	nonce := tx.Nonce()
	s.metrics.resubmitTransactionTotal.WithLabelValues(s.service, s.name).Inc()
	signedTx, err := s.createTx(&feeData, tx.To(), tx.Data(), tx.BlobTxSidecar(), nonce)
	if err != nil {
		log.Error("failed to create signed tx (resubmit case)", "from", s.transactionSigner.GetAddr().String(), "nonce", nonce, "err", err)
		return nil, err
	}
	return signedTx, nil
}

// checkPendingTransaction checks the confirmation status of pending transactions against the latest confirmed block number.
// If a transaction hasn't been confirmed after a certain number of blocks, it will be resubmitted with an increased gas price.
func (s *Sender) checkPendingTransaction() {
	s.metrics.senderCheckPendingTransactionTotal.WithLabelValues(s.service, s.name).Inc()

	blockNumber, baseFee, blobBaseFee, err := s.getBlockNumberAndBaseFeeAndBlobFee(s.ctx)
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
		originalTx := new(gethTypes.Transaction)
		if err := originalTx.DecodeRLP(rlp.NewStream(bytes.NewReader(txnToCheck.RLPEncoding), 0)); err != nil {
			log.Error("failed to decode RLP", "context ID", txnToCheck.ContextID, "sender meta", s.getSenderMeta(), "err", err)
			continue
		}

		receipt, err := s.client.TransactionReceipt(s.ctx, originalTx.Hash())
		if err == nil { // tx confirmed.
			if receipt.BlockNumber.Uint64() <= confirmed {
				if dbTxErr := s.db.Transaction(func(dbTX *gorm.DB) error {
					// Update the status of the transaction to TxStatusConfirmed.
					if updateErr := s.pendingTransactionOrm.UpdateTransactionStatusByTxHash(s.ctx, originalTx.Hash(), types.TxStatusConfirmed, dbTX); updateErr != nil {
						log.Error("failed to update transaction status by tx hash", "hash", originalTx.Hash().String(), "sender meta", s.getSenderMeta(), "from", s.transactionSigner.GetAddr().String(), "nonce", originalTx.Nonce(), "err", updateErr)
						return updateErr
					}
					// Update other transactions with the same nonce and sender address as failed.
					if updateErr := s.pendingTransactionOrm.UpdateOtherTransactionsAsFailedByNonce(s.ctx, txnToCheck.SenderAddress, originalTx.Nonce(), originalTx.Hash(), dbTX); updateErr != nil {
						log.Error("failed to update other transactions as failed by nonce", "senderAddress", txnToCheck.SenderAddress, "nonce", originalTx.Nonce(), "excludedTxHash", originalTx.Hash(), "err", updateErr)
						return updateErr
					}
					return nil
				}); dbTxErr != nil {
					log.Error("db transaction failed after receiving confirmation", "err", dbTxErr)
					return
				}

				// send confirm message
				s.confirmCh <- &Confirmation{
					ContextID:    txnToCheck.ContextID,
					IsSuccessful: receipt.Status == gethTypes.ReceiptStatusSuccessful,
					TxHash:       originalTx.Hash(),
					SenderType:   s.senderType,
				}
			}
		} else if txnToCheck.Status == types.TxStatusPending && // Only try resubmitting a new transaction based on gas price of the last transaction (status pending) with same ContextID.
			s.config.EscalateBlocks+txnToCheck.SubmitBlockNumber <= blockNumber {

			// blockNumber is the block number with "latest" tag, so we need to check the current nonce of the sender address to ensure that the previous transaction has been confirmed.
			// otherwise it's not very necessary to bump the gas price. Also worth noting is that, during bumping gas prices, the sender would consider the new basefee and blobbasefee of L1.
			currentNonce, err := s.client.NonceAt(s.ctx, common.HexToAddress(txnToCheck.SenderAddress), new(big.Int).SetUint64(blockNumber))
			if err != nil {
				log.Error("failed to get current nonce from node", "address", txnToCheck.SenderAddress, "blockNumber", blockNumber, "err", err)
				return
			}

			// early return if the previous transaction has not been confirmed yet.
			// currentNonce is already the confirmed nonce + 1.
			if originalTx.Nonce() > currentNonce {
				log.Debug("previous transaction not yet confirmed, skip bumping gas price", "address", txnToCheck.SenderAddress, "currentNonce", currentNonce, "txNonce", originalTx.Nonce())
				continue
			}

			// It's possible that the pending transaction was marked as failed earlier in this loop (e.g., if one of its replacements has already been confirmed).
			// Therefore, we fetch the current transaction status again for accuracy before proceeding.
			status, err := s.pendingTransactionOrm.GetTxStatusByTxHash(s.ctx, originalTx.Hash())
			if err != nil {
				log.Error("failed to get transaction status by tx hash", "hash", originalTx.Hash().String(), "err", err)
				return
			}
			if status == types.TxStatusConfirmedFailed {
				log.Warn("transaction already marked as failed, skipping resubmission", "hash", originalTx.Hash().String())
				continue
			}

			log.Info("resubmit transaction",
				"service", s.service,
				"name", s.name,
				"hash", originalTx.Hash().String(),
				"from", s.transactionSigner.GetAddr().String(),
				"nonce", originalTx.Nonce(),
				"submitBlockNumber", txnToCheck.SubmitBlockNumber,
				"currentBlockNumber", blockNumber,
				"escalateBlocks", s.config.EscalateBlocks)

			newSignedTx, err := s.createReplacingTransaction(originalTx, baseFee, blobBaseFee)
			if err != nil {
				s.metrics.resubmitTransactionFailedTotal.WithLabelValues(s.service, s.name).Inc()
				log.Error("failed to resubmit transaction", "context ID", txnToCheck.ContextID, "sender meta", s.getSenderMeta(), "from", s.transactionSigner.GetAddr().String(), "nonce", originalTx.Nonce(), "err", err)
				return
			}

			// Update the status of the original transaction as replaced, while still checking its confirmation status.
			// Insert the new transaction that has replaced the original one, and set the status as pending.
			// A corner case is that the transaction is inserted into the table but not sent to the chain, because the server is stopped in the middle.
			// This case will be handled by the checkPendingTransaction function.
			if dbTxErr := s.db.Transaction(func(dbTX *gorm.DB) error {
				if updateErr := s.pendingTransactionOrm.UpdateTransactionStatusByTxHash(s.ctx, originalTx.Hash(), types.TxStatusReplaced, dbTX); updateErr != nil {
					return fmt.Errorf("failed to update status of transaction with hash %s to TxStatusReplaced, err: %w", newSignedTx.Hash().String(), updateErr)
				}
				if updateErr := s.pendingTransactionOrm.InsertPendingTransaction(s.ctx, txnToCheck.ContextID, s.getSenderMeta(), newSignedTx, blockNumber, dbTX); updateErr != nil {
					return fmt.Errorf("failed to insert new pending transaction with context ID: %s, nonce: %d, hash: %v, previous block number: %v, current block number: %v, err: %w", txnToCheck.ContextID, newSignedTx.Nonce(), newSignedTx.Hash().String(), txnToCheck.SubmitBlockNumber, blockNumber, updateErr)
				}
				return nil
			}); dbTxErr != nil {
				log.Error("db transaction failed after resubmitting", "err", dbTxErr)
				return
			}

			if err := s.client.SendTransaction(s.ctx, newSignedTx); err != nil {
				// SendTransaction failed, need to rollback the previous database changes
				if rollbackErr := s.db.Transaction(func(tx *gorm.DB) error {
					// Restore original transaction status back to pending
					if updateErr := s.pendingTransactionOrm.UpdateTransactionStatusByTxHash(s.ctx, originalTx.Hash(), types.TxStatusPending, tx); updateErr != nil {
						return fmt.Errorf("failed to rollback status of original transaction, err: %w", updateErr)
					}
					// Delete the new transaction that was inserted
					if updateErr := s.pendingTransactionOrm.DeleteTransactionByTxHash(s.ctx, newSignedTx.Hash(), tx); updateErr != nil {
						return fmt.Errorf("failed to delete new transaction, err: %w", updateErr)
					}
					return nil
				}); rollbackErr != nil {
					// Both SendTransaction and rollback failed
					log.Error("failed to rollback database after SendTransaction failed", "tx hash", newSignedTx.Hash().String(), "from", s.transactionSigner.GetAddr().String(), "nonce", newSignedTx.Nonce(), "sendTxErr", err, "rollbackErr", rollbackErr)
					return
				}

				log.Error("failed to send replacing tx", "tx hash", newSignedTx.Hash().String(), "from", s.transactionSigner.GetAddr().String(), "nonce", newSignedTx.Nonce(), "err", err)
				return
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
		Address: s.transactionSigner.GetAddr(),
		Type:    s.senderType,
	}
}

func (s *Sender) getBlockNumberAndBaseFeeAndBlobFee(ctx context.Context) (uint64, uint64, uint64, error) {
	header, err := s.client.HeaderByNumber(ctx, big.NewInt(rpc.PendingBlockNumber.Int64()))
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get header by number, err: %w", err)
	}

	var baseFee uint64
	if header.BaseFee != nil {
		baseFee = header.BaseFee.Uint64()
	}

	var blobBaseFee uint64
	if excess := header.ExcessBlobGas; excess != nil {
		blobBaseFee = eip4844.CalcBlobFee(*excess).Uint64()
	}
	// header.Number.Uint64() returns the pendingBlockNumber, so we minus 1 to get the latestBlockNumber.
	return header.Number.Uint64() - 1, baseFee, blobBaseFee, nil
}

func makeSidecar(blob *kzg4844.Blob) (*gethTypes.BlobTxSidecar, error) {
	if blob == nil {
		return nil, errors.New("blob cannot be nil")
	}

	blobs := []kzg4844.Blob{*blob}
	var commitments []kzg4844.Commitment
	var proofs []kzg4844.Proof

	for i := range blobs {
		c, err := kzg4844.BlobToCommitment(&blobs[i])
		if err != nil {
			return nil, fmt.Errorf("failed to get blob commitment, err: %w", err)
		}

		p, err := kzg4844.ComputeBlobProof(&blobs[i], c)
		if err != nil {
			return nil, fmt.Errorf("failed to compute blob proof, err: %w", err)
		}

		commitments = append(commitments, c)
		proofs = append(proofs, p)
	}

	return &gethTypes.BlobTxSidecar{
		Blobs:       blobs,
		Commitments: commitments,
		Proofs:      proofs,
	}, nil
}
