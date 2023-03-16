package l2

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
	"golang.org/x/sync/errgroup"
	"modernc.org/mathutil"

	"scroll-tech/common/utils"

	"scroll-tech/common/types"

	bridge_abi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/sender"
)

const processMsgLimit = 100

func (r *Layer2Relayer) checkSubmittedMessages() error {
	var nonce uint64
	for {
		// msgs are sorted by nonce in increasing order
		msgs, err := r.db.GetL2Messages(
			map[string]interface{}{"status": types.MsgSubmitted},
			fmt.Sprintf("AND nonce > %d", nonce),
			fmt.Sprintf("ORDER BY nonce ASC LIMIT %d", processMsgLimit),
		)
		if err != nil || len(msgs) == 0 {
			return err
		}

		var batch *types.BlockBatch
		for msg := msgs[0]; len(msgs) > 0; { //nolint:staticcheck
			// Wait until sender's pending is not full.
			utils.TryTimes(-1, func() bool {
				return !r.messageSender.IsFull()
			})
			msg, msgs = msgs[0], msgs[1:]
			nonce = mathutil.MaxUint64(nonce, msg.Nonce)

			// Get batch by block number.
			if batch == nil || msg.Height < batch.StartBlockNumber || msg.Height > batch.EndBlockNumber {
				batches, err := r.db.GetBlockBatches(
					map[string]interface{}{},
					fmt.Sprintf("AND start_block_number <= %d AND end_block_number >= %d", msg.Height, msg.Height),
				)
				// If get batch failed, stop and return immediately.
				if err != nil || len(batches) == 0 {
					return err
				}
				batch = batches[0]
			}

			data, err := r.packRelayMessage(msg, common.HexToHash(batch.Hash))
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
				log.Error("failed to load or send l2 submitted tx", "batch hash", batch.Hash, "msg hash", msg.MsgHash, "err", err)
			} else {
				r.processingMessage.Store(msg.MsgHash, msg.MsgHash)
			}
		}
	}
}

// ProcessSavedEvents relays saved un-processed cross-domain transactions to desired blockchain
func (r *Layer2Relayer) ProcessSavedEvents() {
	batch, err := r.db.GetLatestFinalizedBatch()
	if err != nil {
		log.Error("GetLatestFinalizedBatch failed", "err", err)
		return
	}

	// msgs are sorted by nonce in increasing order
	msgs, err := r.db.GetL2Messages(
		map[string]interface{}{"status": types.MsgPending},
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
				return r.processSavedEvent(msg)
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

func (r *Layer2Relayer) packRelayMessage(msg *types.L2Message, batchHash common.Hash) ([]byte, error) {
	// TODO: rebuild the withdraw trie to generate the merkle proof
	proof := bridge_abi.IL1ScrollMessengerL2MessageProof{
		BatchHash:   batchHash,
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
	msgNonce := big.NewInt(int64(msg.Nonce))
	calldata := common.Hex2Bytes(msg.Calldata)
	data, err := r.l1MessengerABI.Pack("relayMessageWithProof", from, target, value, msgNonce, calldata, proof)
	if err != nil {
		log.Error("Failed to pack relayMessageWithProof", "msg.nonce", msg.Nonce, "err", err)
		// TODO: need to skip this message by changing its status to MsgError
		return nil, err
	}
	return data, nil
}

func (r *Layer2Relayer) processSavedEvent(msg *types.L2Message) error {
	// @todo fetch merkle proof from l2geth
	log.Info("Processing L2 Message", "msg.nonce", msg.Nonce, "msg.height", msg.Height)

	// Get the block info that contains the message
	blockInfos, err := r.db.GetL2BlockInfos(map[string]interface{}{"number": msg.Height})
	if err != nil {
		log.Error("Failed to GetL2BlockInfos from DB", "number", msg.Height)
	}
	blockInfo := blockInfos[0]
	if !blockInfo.BatchHash.Valid {
		log.Error("Block has not been batched yet", "number", blockInfo.Number, "msg.nonce", msg.Nonce)
		return nil
	}

	data, err := r.packRelayMessage(msg, common.HexToHash(blockInfo.BatchHash.String))
	if err != nil {
		log.Error("Failed to pack relayMessageWithProof", "msg.nonce", msg.Nonce, "err", err)
		// TODO: need to skip this message by changing its status to MsgError
		return err
	}

	hash, err := r.messageSender.SendTransaction(msg.MsgHash, &r.cfg.MessengerContractAddress, big.NewInt(0), data)
	if err != nil && err.Error() == "execution reverted: Message expired" {
		return r.db.UpdateLayer2Status(r.ctx, msg.MsgHash, types.MsgExpired)
	}
	if err != nil && err.Error() == "execution reverted: Message successfully executed" {
		return r.db.UpdateLayer2Status(r.ctx, msg.MsgHash, types.MsgConfirmed)
	}
	if err != nil {
		if !errors.Is(err, sender.ErrNoAvailableAccount) {
			log.Error("Failed to send relayMessageWithProof tx to layer1 ", "msg.height", msg.Height, "msg.MsgHash", msg.MsgHash, "err", err)
		}
		return err
	}
	bridgeL2MsgsRelayedTotalCounter.Inc(1)
	log.Info("relayMessageWithProof to layer1", "msgHash", msg.MsgHash, "txhash", hash.String())

	// save status in db
	// @todo handle db error
	err = r.db.UpdateLayer2StatusAndLayer1Hash(r.ctx, msg.MsgHash, types.MsgSubmitted, hash.String())
	if err != nil {
		log.Error("UpdateLayer2StatusAndLayer1Hash failed", "msgHash", msg.MsgHash, "err", err)
		return err
	}
	r.processingMessage.Store(msg.MsgHash, msg.MsgHash)
	return nil
}
