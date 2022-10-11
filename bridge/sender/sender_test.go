package sender

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/math"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/stretchr/testify/assert"

	"scroll-tech/internal/mock/handler"

	"scroll-tech/bridge/config"
)

var (
	testSenderConfig         config.SenderConfig
	testChainID              *big.Int
	testTxValue              *big.Int
	testStartNonce           uint64
	testStartBlock           uint64
	testMaxFeePerGas         uint64
	testMaxPriorityFeePerGas uint64
	testGasLimit             uint64
)

func init() {
	testSenderConfig = DefaultSenderConfig
	testSenderConfig.TxType = DynamicFeeTxType
	testChainID = new(big.Int).SetUint64(1)
	testStartNonce = 100
	testStartBlock = 123
	testMaxFeePerGas = 333
	testMaxPriorityFeePerGas = 444
	testTxValue = big.NewInt(233)
	testGasLimit = 555666
}

func clearChan(ch chan *Confirmation) {
	for {
		if len(ch) == 0 {
			break
		}
		r := <-ch
		log.Info("confirmed", r)
	}
}

func setupSender() (*Sender, *handler.GethRPCHandler, chan *Confirmation) {
	handler := handler.NewGethRPCHanlder(testChainID)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimSpace(r.URL.Path) {
		case "/":
			handler.Handle(w, r)
		case "/ws":
			handler.WsHandle(w, r)
		default:
			http.NotFoundHandler().ServeHTTP(w, r)
		}
	}))

	testSenderConfig.Endpoint = "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	prv, err := crypto.GenerateKey()
	if err != nil {
		return nil, nil, nil
	}

	address := crypto.PubkeyToAddress(prv.PublicKey)
	handler.UpdateNonce(address, testStartNonce)
	handler.UpdateBlockNumber(testStartBlock)
	handler.UpdateMaxFeePerGas(testMaxFeePerGas)
	handler.UpdateMaxPriorityFeePerGas(testMaxPriorityFeePerGas)
	handler.UpdateGasLimit(testGasLimit)

	ch := make(chan *Confirmation, 3)
	sender, err := NewSender(context.Background(), ch, testSenderConfig, prv)
	if err != nil {
		return nil, nil, nil
	}

	return sender, handler, ch
}

func TestInitializeCorrectly(t *testing.T) {
	sender, _, _ := setupSender()
	if sender == nil {
		t.Fatalf("setup sender failed")
	}
	defer sender.Stop()

	if sender.nonce != testStartNonce {
		t.Fatalf("Invalid nonce, want %d, got %d", testStartNonce, sender.nonce)
	}

	if sender.pendingNonce != testStartNonce {
		t.Fatalf("Invalid pending nonce, want %d, got %d", testStartNonce, sender.pendingNonce)
	}

	if sender.blockNumber.Cmp(big.NewInt(int64(testStartBlock))) != 0 {
		t.Fatalf("Invalid blockNumber, want %d, got %s", testStartBlock, sender.blockNumber.Text(10))
	}
}

func TestStopNormally(t *testing.T) {
	sender, _, _ := setupSender()
	if sender == nil {
		t.Fatalf("setup sender failed")
	}
	sender.Stop()
}

func TestNonceBumpOnNormalCase(t *testing.T) {
	sender, handler, ch := setupSender()
	defer clearChan(ch)
	if sender == nil {
		t.Fatalf("setup sender failed")
	}
	defer sender.Stop()

	sender.SendTransaction(&common.Address{}, testTxValue, make([]byte, 0))
	if sender.nonce != testStartNonce+1 {
		t.Fatalf("Invalid nonce, want %d, got %d", testStartNonce+1, sender.nonce)
	}
	if sender.pendingNonce != testStartNonce {
		t.Fatalf("Invalid pending nonce, want %d, got %d", testStartNonce, sender.pendingNonce)
	}

	// check tx status
	pending, ok := sender.pendingTxns[sender.pendingNonce]
	if !ok {
		t.Fatalf("Missing pending txn")
	}

	if pending.submitAt.Cmp(big.NewInt(int64(testStartBlock))) != 0 {
		t.Fatalf("Invalid submit at, want %d, got %s", testStartBlock, pending.submitAt.Text(10))
	}
	if pending.tx.ChainId().Cmp(testChainID) != 0 {
		t.Fatalf("Invalid tx chain id, want %s, got %s", testChainID.Text(10), pending.tx.ChainId().Text(10))
	}
	if pending.tx.Value().Cmp(testTxValue) != 0 {
		t.Fatalf("Invalid tx value, want %s, got %s", testTxValue.Text(10), pending.tx.Value().Text(10))
	}
	if pending.tx.Gas() != testGasLimit*15/10 {
		t.Fatalf("Invalid tx gas limit, want %d, got %d", testGasLimit, pending.tx.Gas())
	}
	gasTipCap, err := sender.client.SuggestGasTipCap(sender.ctx)
	assert.NoError(t, err)
	gasFeeCap := math.BigMax(big.NewInt(int64(testMaxFeePerGas)), big.NewInt(1000000000))
	gasFeeCap = math.BigMax(gasFeeCap, gasTipCap)
	gasTipCap = math.BigMin(gasFeeCap, gasTipCap)
	if pending.tx.GasFeeCap().Cmp(gasFeeCap) != 0 {
		t.Fatalf("Invalid tx gas fee cap, want %s, got %s", gasFeeCap.Text(10), pending.tx.GasPrice().Text(10))
	}
	if pending.tx.GasTipCap().Cmp(gasTipCap) != 0 {
		t.Fatalf("Invalid tx gas tip cap, want %s, got %s", gasTipCap.Text(10), pending.tx.GasPrice().Text(10))
	}
	if pending.tx.GasPrice().Cmp(pending.tx.GasFeeCap()) != 0 {
		t.Fatalf("Invalid tx gas price, want %s, got %s", pending.tx.GasFeeCap().Text(10), pending.tx.GasPrice().Text(10))
	}
	_, ok = handler.Txns[pending.tx.Hash()]
	if !ok {
		t.Fatalf("Missing txn in server")
	}

	// bump on chain nonce
	handler.UpdateNonce(sender.address, testStartNonce+1)

	// advance block
	sender.blockNumber = sender.blockNumber.Add(sender.blockNumber, big.NewInt(1))
	sender.CheckPendingTransaction(ch, big.NewInt(10))

	// should confirmed now
	if sender.pendingNonce != testStartNonce+1 {
		t.Fatalf("Invalid pending nonce, want %d, got %d", testStartNonce+1, sender.pendingNonce)
	}
}

func TestEscalate(t *testing.T) {
	sender, handler, ch := setupSender()
	defer clearChan(ch)
	if sender == nil {
		t.Fatalf("setup sender failed")
	}
	defer sender.Stop()

	sender.SendTransaction(&common.Address{}, testTxValue, make([]byte, 0))
	if sender.nonce != testStartNonce+1 {
		t.Fatalf("Invalid nonce, want %d, got %d", testStartNonce+1, sender.nonce)
	}
	if sender.pendingNonce != testStartNonce {
		t.Fatalf("Invalid pending nonce, want %d, got %d", testStartNonce, sender.pendingNonce)
	}

	// advance block
	sender.blockNumber = sender.blockNumber.Add(sender.blockNumber, big.NewInt(10))
	sender.CheckPendingTransaction(ch, big.NewInt(3))

	// nonce should not change
	if sender.nonce != testStartNonce+1 {
		t.Fatalf("Invalid nonce, want %d, got %d", testStartNonce+1, sender.nonce)
	}
	if sender.pendingNonce != testStartNonce {
		t.Fatalf("Invalid pending nonce, want %d, got %d", testStartNonce, sender.pendingNonce)
	}

	// check tx status
	pending, ok := sender.pendingTxns[sender.pendingNonce]
	if !ok {
		t.Fatalf("Missing pending txn")
	}
	if pending.submitAt.Cmp(sender.blockNumber) != 0 {
		t.Fatalf("Invalid submit at, want %s, got %s", sender.blockNumber.Text(10), pending.submitAt.Text(10))
	}
	if pending.tx.ChainId().Cmp(testChainID) != 0 {
		t.Fatalf("Invalid tx chain id, want %s, got %s", testChainID.Text(10), pending.tx.ChainId().Text(10))
	}
	if pending.tx.Value().Cmp(testTxValue) != 0 {
		t.Fatalf("Invalid tx value, want %s, got %s", testTxValue.Text(10), pending.tx.Value().Text(10))
	}
	if pending.tx.Gas() != testGasLimit*15/10 {
		t.Fatalf("Invalid tx gas limit, want %d, got %d", testGasLimit, pending.tx.Gas())
	}
	gasTipCap, err := sender.client.SuggestGasTipCap(sender.ctx)
	assert.NoError(t, err)
	// Make sure feeCap is bigger than txpool's gas price. 1000000000 is l2geth's default pool.gas value.
	gasFeeCap := math.BigMax(big.NewInt(int64(testMaxFeePerGas)), big.NewInt(1000000000))
	gasFeeCap = math.BigMax(gasFeeCap, gasTipCap)
	gasTipCap = math.BigMin(gasFeeCap, gasTipCap)
	gasFeeCap = gasFeeCap.Mul(gasFeeCap, big.NewInt(int64(testSenderConfig.EscalateMultipleNum)))
	gasFeeCap = gasFeeCap.Div(gasFeeCap, big.NewInt(int64(testSenderConfig.EscalateMultipleDen)))
	if pending.tx.GasFeeCap().Cmp(gasFeeCap) != 0 {
		t.Fatalf("Invalid tx gas fee cap, want %s, got %s", gasFeeCap.Text(10), pending.tx.GasPrice().Text(10))
	}
	gasTipCap = gasTipCap.Mul(gasTipCap, big.NewInt(int64(testSenderConfig.EscalateMultipleNum)))
	gasTipCap = gasTipCap.Div(gasTipCap, big.NewInt(int64(testSenderConfig.EscalateMultipleDen)))
	if pending.tx.GasTipCap().Cmp(gasTipCap) != 0 {
		t.Fatalf("Invalid tx gas tip cap, want %s, got %s", gasTipCap.Text(10), pending.tx.GasPrice().Text(10))
	}
	if pending.tx.GasPrice().Cmp(pending.tx.GasFeeCap()) != 0 {
		t.Fatalf("Invalid tx gas price, want %s, got %s", pending.tx.GasFeeCap().Text(10), pending.tx.GasPrice().Text(10))
	}

	_, ok = handler.Txns[pending.tx.Hash()]
	if !ok {
		t.Fatalf("Missing txn in server")
	}
}
