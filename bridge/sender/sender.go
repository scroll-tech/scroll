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

	geth "github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/math"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/viper"
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
	client  *ethclient.Client // The client to retrieve on chain data or send transaction.
	chainID *big.Int          // The chain id of the endpoint
	ctx     context.Context

	// sender config
	vp *viper.Viper

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
func NewSender(ctx context.Context, vp *viper.Viper, privs []*ecdsa.PrivateKey) (*Sender, error) {
	client, err := ethclient.Dial(vp.GetString("endpoint"))
	if err != nil {
		return nil, err
	}

	// get chainID from client
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, err
	}

	minBalance := vp.GetBigInt("min_balance")

	auths, err := newAccountPool(ctx, minBalance, client, privs)
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
		client:        client,
		chainID:       chainID,
		vp:            vp,
		auths:         auths,
		confirmCh:     make(chan *Confirmation, 128),
		blockNumber:   header.Number.Uint64(),
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

// NumberOfAccounts return the count of accounts.
func (s *Sender) NumberOfAccounts() int {
	return len(s.auths.accounts)
}

func (s *Sender) getFeeData(auth *bind.TransactOpts, target *common.Address, value *big.Int, data []byte) (*FeeData, error) {
	// estimate gas limit
	gasLimit, err := s.client.EstimateGas(s.ctx, geth.CallMsg{From: auth.From, To: target, Value: value, Data: data})
	if err != nil {
		return nil, err
	}
	gasLimit = gasLimit * 15 / 10 // 50% extra gas to void out of gas error
	// @todo change it when Scroll enable EIP1559
	txType := s.vp.GetString("tx_type")
	if txType != DynamicFeeTxType {
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
	if feeData, err = s.getFeeData(auth, target, value, data); err != nil {
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
	txType := s.vp.GetString("tx_type")
	switch txType {
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
	escalateMultipleNum := new(big.Int).SetUint64(uint64(s.vp.GetInt("escalate_multiple_num")))
	escalateMultipleDen := new(big.Int).SetUint64(uint64(s.vp.GetInt("escalate_multiple_den")))
	maxGasPrice := s.vp.GetBigInt("max_gas_price")

	txType := s.vp.GetString("tx_type")
	switch txType {
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

// CheckPendingTransaction Check pending transaction given number of blocks to wait before confirmation.
func (s *Sender) CheckPendingTransaction(header *types.Header) {
	number := header.Number.Uint64()
	atomic.StoreUint64(&s.blockNumber, number)
	atomic.StoreUint64(&s.baseFeePerGas, header.BaseFee.Uint64())
	s.pendingTxs.Range(func(key, value interface{}) bool {
		// ignore empty id, since we use empty id to occupy pending task
		if value == nil || reflect.ValueOf(value).IsNil() {
			return true
		}

		pending := value.(*PendingTransaction)
		receipt, err := s.client.TransactionReceipt(s.ctx, pending.tx.Hash())
		escalateBlocks := uint64(s.vp.GetInt("escalate_blocks"))
		if (err == nil) && (receipt != nil) {
			confirmations := uint64(s.vp.GetInt("confirmations"))
			if number >= receipt.BlockNumber.Uint64()+confirmations {
				s.pendingTxs.Delete(key)
				// send confirm message
				s.confirmCh <- &Confirmation{
					ID:           pending.id,
					IsSuccessful: receipt.Status == types.ReceiptStatusSuccessful,
					TxHash:       pending.tx.Hash(),
				}
			}
		} else if escalateBlocks+pending.submitAt < number {
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
	checkPendingTimeSec := s.vp.GetInt("check_pending_time_sec")
	checkTick := time.NewTicker(time.Duration(checkPendingTimeSec) * time.Second)
	defer checkTick.Stop()

	checkBalanceTimeMin := s.vp.GetInt("check_balance_time_min")
	checkBalanceTicker := time.NewTicker(time.Duration(checkBalanceTimeMin) * time.Minute)
	defer checkBalanceTicker.Stop()

	for {
		select {
		case <-checkTick.C:
			header, err := s.client.HeaderByNumber(s.ctx, nil)
			if err != nil {
				log.Error("failed to get latest head", "err", err)
				continue
			}
			s.CheckPendingTransaction(header)
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
