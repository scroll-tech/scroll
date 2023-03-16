package l1

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
	"modernc.org/mathutil"

	"scroll-tech/common/types"
	"scroll-tech/common/utils"

	"scroll-tech/bridge/sender"
)

func (r *Layer1Relayer) checkSubmittedMessages() error {
	var (
		index    uint64
		msgsSize = 100
	)
	for {
		msgs, err := r.db.GetL1Messages(
			map[string]interface{}{"status": types.MsgSubmitted},
			fmt.Sprintf("AND queue_index > %d", index),
			fmt.Sprintf("ORDER BY queue_index ASC LIMIT %d", msgsSize),
		)
		if err != nil || len(msgs) == 0 {
			return err
		}

		for msg := msgs[0]; len(msgs) > 0; { //nolint:staticcheck
			// If pending txs pool is full, wait until pending pool is available.
			utils.TryTimes(-1, func() bool {
				return !r.messageSender.IsFull()
			})
			msg, msgs = msgs[0], msgs[1:]
			index = mathutil.MaxUint64(index, msg.QueueIndex)

			if err = r.messageSender.LoadOrSendTx(
				common.HexToHash(msg.Layer2Hash),
				msg.MsgHash,
				&r.cfg.MessengerContractAddress,
				big.NewInt(0),
				common.Hex2Bytes(msg.Calldata),
			); err != nil {
				log.Error("failed to load or send l1 submitted tx", "msg hash", msg.MsgHash, "err", err)
				return err
			}
		}
	}
}

// ProcessSavedEvents relays saved un-processed cross-domain transactions to desired blockchain
func (r *Layer1Relayer) ProcessSavedEvents() {
	// msgs are sorted by nonce in increasing order
	msgs, err := r.db.GetL1MessagesByStatus(types.MsgPending, 100)
	if err != nil {
		log.Error("Failed to fetch unprocessed L1 messages", "err", err)
		return
	}

	if len(msgs) > 0 {
		log.Info("Processing L1 messages", "count", len(msgs))
	}

	for _, msg := range msgs {
		if err = r.processSavedEvent(msg); err != nil {
			if !errors.Is(err, sender.ErrNoAvailableAccount) {
				log.Error("failed to process event", "msg.msgHash", msg.MsgHash, "err", err)
			}
			return
		}
	}
}

func (r *Layer1Relayer) processSavedEvent(msg *types.L1Message) error {
	calldata := common.Hex2Bytes(msg.Calldata)

	hash, err := r.messageSender.SendTransaction(msg.MsgHash, &r.cfg.MessengerContractAddress, big.NewInt(0), calldata)
	if err != nil && err.Error() == "execution reverted: Message expired" {
		return r.db.UpdateLayer1Status(r.ctx, msg.MsgHash, types.MsgExpired)
	}
	if err != nil && err.Error() == "execution reverted: Message successfully executed" {
		return r.db.UpdateLayer1Status(r.ctx, msg.MsgHash, types.MsgConfirmed)
	}
	if err != nil {
		return err
	}
	bridgeL1MsgsRelayedTotalCounter.Inc(1)
	log.Info("relayMessage to layer2", "msg hash", msg.MsgHash, "tx hash", hash)

	err = r.db.UpdateLayer1StatusAndLayer2Hash(r.ctx, msg.MsgHash, types.MsgSubmitted, hash.String())
	if err != nil {
		log.Error("UpdateLayer1StatusAndLayer2Hash failed", "msg.msgHash", msg.MsgHash, "msg.height", msg.Height, "err", err)
	}
	return err
}
