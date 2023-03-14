package sender

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/bridge/utils"

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
	// ErrNoAvailableAccount indicates no available account error in the account pool.
	ErrNoAvailableAccount = errors.New("sender has no available account to send transaction")
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

	// account fields.
	auths *accountPool

	blockNumber   uint64   // Current block number on chain.
	baseFeePerGas uint64   // Current base fee per gas on chain
	pendingTxs    sync.Map // Mapping from nonce to pending transaction
	confirmCh     chan *Confirmation

	stopCh chan struct{}
}

// NewSender returns a new instance of transaction sender
// txConfirmationCh is used to notify confirmed transaction
func NewSender(ctx context.Context, config *config.SenderConfig, privs []*ecdsa.PrivateKey) (*Sender, error) {
	client, err := ethclient.Dial(config.Endpoint)
	if err != nil {
		return nil, err
	}

	// get chainID from client
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, err
	}

	auths, err := newAccountPool(ctx, config.MinBalance, client, privs)
	if err != nil {
		return nil, fmt.Errorf("failed to create account pool, err: %v", err)
	}

	// get header by number
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}

	var baseFeePerGas uint64
	if config.TxType == DynamicFeeTxType {
		if header.BaseFee != nil {
			baseFeePerGas = header.BaseFee.Uint64()
		} else {
			return nil, errors.New("DynamicFeeTxType not supported, header.BaseFee nil")
		}
	}

	sender := &Sender{
		ctx:           ctx,
		config:        config,
		client:        client,
		chainID:       chainID,
		auths:         auths,
		confirmCh:     make(chan *Confirmation, 128),
		blockNumber:   header.Number.Uint64(),
		baseFeePerGas: baseFeePerGas,
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

// NumberOfAccounts return the count of accounts.
func (s *Sender) NumberOfAccounts() int {
	return len(s.auths.accounts)
}

func (s *Sender) getFeeData(auth *bind.TransactOpts, target *common.Address, value *big.Int, data []byte, minGasLimit uint64) (*FeeData, error) {
	if s.config.TxType == DynamicFeeTxType {
		return s.estimateDynamicGas(auth, target, value, data, minGasLimit)
	}
	return s.estimateLegacyGas(auth, target, value, data, minGasLimit)
}

// SendTransaction send a signed L2tL1 transaction.
func (s *Sender) SendTransaction(ID string, target *common.Address, value *big.Int, data []byte, minGasLimit uint64) (hash common.Hash, err error) {
	// We occupy the ID, in case some other threads call with the same ID in the same time
	if _, loaded := s.pendingTxs.LoadOrStore(ID, nil); loaded {
		return common.Hash{}, fmt.Errorf("has the repeat tx ID, ID: %s", ID)
	}
	// get
	auth := s.auths.getAccount()
	if auth == nil {
		s.pendingTxs.Delete(ID) // release the ID on failure
		return common.Hash{}, ErrNoAvailableAccount
	}

	defer s.auths.releaseAccount(auth)
	defer func() {
		if err != nil {
			s.pendingTxs.Delete(ID) // release the ID on failure
		}
	}()

	var (
		feeData *FeeData
		tx      *types.Transaction
	)
	// estimate gas fee
	if feeData, err = s.getFeeData(auth, target, value, data, minGasLimit); err != nil {
		return
	}
	if tx, err = s.createAndSendTx(auth, feeData, target, value, data, nil); err == nil {
		// add pending transaction to queue
		pending := &PendingTransaction{
			tx:       tx,
			id:       ID,
			signer:   auth,
			submitAt: atomic.LoadUint64(&s.blockNumber),
			feeData:  feeData,
		}
		s.pendingTxs.Store(ID, pending)
		return tx.Hash(), nil
	}

	return
}

func (s *Sender) createAndSendTx(auth *bind.TransactOpts, feeData *FeeData, target *common.Address, value *big.Int, data []byte, overrideNonce *uint64) (tx *types.Transaction, err error) {
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
	tx, err = auth.Signer(auth.From, types.NewTx(txData))
	if err != nil {
		log.Error("failed to sign tx", "err", err)
		return
	}
	if err = s.client.SendTransaction(s.ctx, tx); err != nil {
		log.Error("failed to send tx", "tx hash", tx.Hash().String(), "err", err)
		// Check if contain nonce, and reset nonce
		// only reset nonce when it is not from resubmit
		if strings.Contains(err.Error(), "nonce") && overrideNonce == nil {
			s.auths.resetNonce(context.Background(), auth)
		}
		return
	}

	// update nonce when it is not from resubmit
	if overrideNonce == nil {
		auth.Nonce = big.NewInt(int64(nonce + 1))
	}
	return
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
		if gasFeeCap.Cmp(maxGasPrice) > 0 {
			gasFeeCap = maxGasPrice
		}
		feeData.gasFeeCap = gasFeeCap
		feeData.gasTipCap = gasTipCap
	}

	nonce := tx.Nonce()
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

	s.pendingTxs.Range(func(key, value interface{}) bool {
		// ignore empty id, since we use empty id to occupy pending task
		if value == nil || reflect.ValueOf(value).IsNil() {
			return true
		}

		pending := value.(*PendingTransaction)
		receipt, err := s.client.TransactionReceipt(s.ctx, pending.tx.Hash())
		if (err == nil) && (receipt != nil) {
			if receipt.BlockNumber.Uint64() <= confirmed {
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
					s.pendingTxs.Delete(key)
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
		return true
	})
}

// Loop is the main event loop
func (s *Sender) loop(ctx context.Context) {
	checkTick := time.NewTicker(time.Duration(s.config.CheckPendingTime) * time.Second)
	defer checkTick.Stop()

	checkBalanceTicker := time.NewTicker(time.Minute * 10)
	defer checkBalanceTicker.Stop()

	for {
		select {
		case <-checkTick.C:
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
			// Check and set balance.
			_ = s.auths.checkAndSetBalances(ctx)
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		}
	}
}
