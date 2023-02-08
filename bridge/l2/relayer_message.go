package l2

import (
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
	"golang.org/x/sync/errgroup"
	"modernc.org/mathutil"

	bridge_abi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/sender"

	"scroll-tech/database/orm"
)

const processMsgLimit = 100

func (r *Layer2Relayer) checkSubmittedMessages() error {
	batch, err := r.db.GetLatestFinalizedBatch()
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if batch == nil {
		return nil
	}

	// msgs are sorted by nonce in increasing order
	msgs, err := r.db.GetL2Messages(
		map[string]interface{}{"status": orm.MsgSubmitted},
		fmt.Sprintf("AND height<=%d", batch.EndBlockNumber),
		fmt.Sprintf("ORDER BY height DESC LIMIT %d", r.messageSender.PendingLimit()),
	)
	if err != nil || len(msgs) == 0 {
		return err
	}

	for _, msg := range msgs {
		data, err := r.messagePack(msg, batch.Index)
		if err != nil {
			continue
		}
		err = r.messageSender.LoadOrSendTx(
			common.HexToHash(msg.Layer1Hash),
			msg.MsgHash,
			&r.cfg.MessengerContractAddress,
			big.NewInt(0),
			data,
		)
		if err != nil {
			log.Error("failed to load or send l2 submitted tx", "batch id", batch.ID, "msg hash", msg.MsgHash, "err", err)
		} else {
			r.processingMessage.Store(msg.MsgHash, msg.MsgHash)
		}
	}
	return nil
}

// ProcessSavedEvents relays saved un-processed cross-domain transactions to desired blockchain
func (r *Layer2Relayer) ProcessSavedEvents(wg *sync.WaitGroup) {
	defer wg.Done()
	batch, err := r.db.GetLatestFinalizedBatch()
	if err != nil {
		log.Error("GetLatestFinalizedBatch failed", "err", err)
		return
	}

	// msgs are sorted by nonce in increasing order
	msgs, err := r.db.GetL2Messages(
		map[string]interface{}{"status": orm.MsgPending},
		fmt.Sprintf("AND height<=%d", batch.EndBlockNumber),
		fmt.Sprintf("ORDER BY nonce ASC LIMIT %d", processMsgLimit),
	)

	if err != nil {
		log.Error("Failed to fetch unprocessed L2 messages", "err", err)
		return
	}

	// process messages in batches
	batchSize := mathutil.Min((runtime.GOMAXPROCS(0)+1)/2, r.messageSender.NumberOfAccounts())
	for size := 0; len(msgs) > 0; msgs = msgs[size:] {
		if size = len(msgs); size > batchSize {
			size = batchSize
		}
		var g errgroup.Group
		for _, msg := range msgs[:size] {
			msg := msg
			g.Go(func() error {
				return r.processSavedEvent(msg, batch.Index)
			})
		}
		if err := g.Wait(); err != nil {
			if !errors.Is(err, sender.ErrNoAvailableAccount) {
				log.Error("failed to process l2 saved event", "err", err)
			}
			return
		}
	}
}

func (r *Layer2Relayer) processSavedEvent(msg *orm.L2Message, index uint64) error {
	data, err := r.messagePack(msg, index)
	if err != nil {
		return err
	}

	hash, err := r.messageSender.SendTransaction(msg.MsgHash, &r.cfg.MessengerContractAddress, big.NewInt(0), data)
	if err != nil && err.Error() == "execution reverted: Message expired" {
		return r.db.UpdateLayer2Status(r.ctx, msg.MsgHash, orm.MsgExpired)
	}
	if err != nil && err.Error() == "execution reverted: Message successfully executed" {
		return r.db.UpdateLayer2Status(r.ctx, msg.MsgHash, orm.MsgConfirmed)
	}
	if err != nil {
		if !errors.Is(err, sender.ErrNoAvailableAccount) {
			log.Error("Failed to send relayMessageWithProof tx to layer1 ", "msg.height", msg.Height, "msg.MsgHash", msg.MsgHash, "err", err)
		}
		return err
	}
	log.Info("relayMessageWithProof to layer1", "msgHash", msg.MsgHash, "txhash", hash.String())

	// save status in db
	// @todo handle db error
	err = r.db.UpdateLayer2StatusAndLayer1Hash(r.ctx, msg.MsgHash, orm.MsgSubmitted, hash.String())
	if err != nil {
		log.Error("UpdateLayer2StatusAndLayer1Hash failed", "msgHash", msg.MsgHash, "err", err)
		return err
	}
	r.processingMessage.Store(msg.MsgHash, msg.MsgHash)
	return nil
}

func (r *Layer2Relayer) messagePack(msg *orm.L2Message, index uint64) ([]byte, error) {
	// @todo fetch merkle proof from l2geth
	log.Info("Processing L2 Message", "msg.nonce", msg.Nonce, "msg.height", msg.Height)

	proof := bridge_abi.IL1ScrollMessengerL2MessageProof{
		BlockHeight: big.NewInt(int64(msg.Height)),
		BatchIndex:  big.NewInt(0).SetUint64(index),
		MerkleProof: make([]byte, 0),
	}
	from := common.HexToAddress(msg.Sender)
	target := common.HexToAddress(msg.Target)
	value, ok := big.NewInt(0).SetString(msg.Value, 10)
	if !ok {
		// @todo maybe panic?
		log.Error("Failed to parse message value", "msg.nonce", msg.Nonce, "msg.height", msg.Height)
		// TODO: need to skip this message by changing its status to MsgError
	}
	fee, _ := big.NewInt(0).SetString(msg.Fee, 10)
	deadline := big.NewInt(int64(msg.Deadline))
	msgNonce := big.NewInt(int64(msg.Nonce))
	calldata := common.Hex2Bytes(msg.Calldata)
	data, err := bridge_abi.L1MessengerMetaABI.Pack("relayMessageWithProof", from, target, value, fee, deadline, msgNonce, calldata, proof)
	if err != nil {
		log.Error("Failed to pack relayMessageWithProof", "msg.nonce", msg.Nonce, "err", err)
		// TODO: need to skip this message by changing its status to MsgError
		return nil, err
	}
	return data, nil
}
