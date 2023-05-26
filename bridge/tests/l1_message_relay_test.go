package tests

import (
	"context"
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	geth_types "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/types"

	"scroll-tech/bridge/internal/controller/relayer"
	"scroll-tech/bridge/internal/controller/watcher"
	"scroll-tech/bridge/internal/orm"
	"scroll-tech/bridge/internal/utils"
)

func testRelayL1MessageSucceed(t *testing.T) {
	db := setupDB(t)
	defer utils.CloseDB(db)

	prepareContracts(t)

	l1Cfg := bridgeApp.Config.L1Config
	l2Cfg := bridgeApp.Config.L2Config

	// Create L1Relayer
	l1Relayer, err := relayer.NewLayer1Relayer(context.Background(), db, l1Cfg.RelayerConfig)
	assert.NoError(t, err)
	// Create L1Watcher
	confirmations := rpc.LatestBlockNumber
	l1Watcher := watcher.NewL1WatcherClient(context.Background(), l1Client, 0, confirmations, l1Cfg.L1MessengerAddress, l1Cfg.L1MessageQueueAddress, l1Cfg.ScrollChainContractAddress, db)

	// Create L2Watcher
	l2Watcher := watcher.NewL2WatcherClient(context.Background(), l2Client, confirmations, l2Cfg.L2MessengerAddress, l2Cfg.L2MessageQueueAddress, l2Cfg.WithdrawTrieRootSlot, db)

	// send message through l1 messenger contract
	nonce, err := l1MessengerInstance.MessageNonce(&bind.CallOpts{})
	assert.NoError(t, err)
	sendTx, err := l1MessengerInstance.SendMessage(l1Auth, l2Auth.From, big.NewInt(0), common.Hex2Bytes("00112233"), big.NewInt(0))
	assert.NoError(t, err)
	sendReceipt, err := bind.WaitMined(context.Background(), l1Client, sendTx)
	assert.NoError(t, err)
	if sendReceipt.Status != geth_types.ReceiptStatusSuccessful || err != nil {
		t.Fatalf("Call failed")
	}

	// l1 watch process events
	l1Watcher.FetchContractEvent()

	l1MessageOrm := orm.NewL1Message(db)
	// check db status
	msg, err := l1MessageOrm.GetL1MessageByQueueIndex(nonce.Uint64())
	assert.NoError(t, err)
	assert.Equal(t, types.MsgStatus(msg.Status), types.MsgPending)
	assert.Equal(t, msg.Target, l2Auth.From.String())

	// process l1 messages
	l1Relayer.ProcessSavedEvents()

	l1Message, err := l1MessageOrm.GetL1MessageByQueueIndex(nonce.Uint64())
	assert.NoError(t, err)
	assert.NotEmpty(t, l1Message.Layer2Hash)
	assert.Equal(t, types.MsgStatus(l1Message.Status), types.MsgSubmitted)

	relayTx, _, err := l2Client.TransactionByHash(context.Background(), common.HexToHash(l1Message.Layer2Hash))
	assert.NoError(t, err)
	relayTxReceipt, err := bind.WaitMined(context.Background(), l2Client, relayTx)
	assert.NoError(t, err)
	assert.Equal(t, len(relayTxReceipt.Logs), 1)

	// fetch message relayed events
	l2Watcher.FetchContractEvent()
	msg, err = l1MessageOrm.GetL1MessageByQueueIndex(nonce.Uint64())
	assert.NoError(t, err)
	assert.Equal(t, types.MsgStatus(msg.Status), types.MsgConfirmed)
}
