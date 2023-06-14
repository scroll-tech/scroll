package message_proof

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"bridge-history-api/db"
	"bridge-history-api/db/orm"
)

type MsgProofUpdater struct {
	ctx          context.Context
	client       *ethclient.Client
	db           db.OrmFactory
	withdrawTrie *WithdrawTrie
}

func NewMsgProofUpdater(ctx context.Context, client *ethclient.Client, confirmations uint64, startBlock uint64, db db.OrmFactory) *MsgProofUpdater {
	return &MsgProofUpdater{
		ctx:          ctx,
		client:       client,
		db:           db,
		withdrawTrie: NewWithdrawTrie(),
	}
}

func (m *MsgProofUpdater) Start() {
	log.Info("MsgProofUpdater Start")
	err := m.initializeWithdrawTrie()
	if err != nil {
		log.Crit("can not initialize withdraw trie", "err", err)
	}
	go func() {
		tick := time.NewTicker(time.Duration(10) * time.Second)
		for {
			select {
			case <-m.ctx.Done():
				tick.Stop()
				return
			case <-tick.C:
				latestBatch, err := m.db.GetLatestBridgeBatch()
				if err != nil {
					log.Error("MsgProofUpdater: Can not get latest BridgeBatch: ", "err", err)
					continue
				}
				if latestBatch == nil {
					continue
				}
				latestBatchHasProof, err := m.db.GetLatestL2SentMsgBactchIndex()
				if err != nil {
					log.Error("MsgProofUpdater: Can not get latest L2SentMsgBatchIndex: ", "err", err)
					continue
				}
				var start uint64
				if latestBatchHasProof < 0 {
					start = 0
				} else {
					start = uint64(latestBatchHasProof) + 1
				}
				if latestBatchHasProof < int64(latestBatch.ID) {
					for i := start; start <= latestBatch.ID; i++ {
						batch, err := m.db.GetBridgeBatchByIndex(i)
						if err != nil {
							log.Error("MsgProofUpdater: Can not get BridgeBatch: ", "err", err)
							break
						}
						// but this should never happen
						if batch == nil {
							log.Error("MsgProofUpdater: No BridgeBatch found: ", "index", i)
							break
						}
						// get all l2 messages in this batch
						msgs, proofs, err := m.appendL2Messages(batch.StartBlockNumber, batch.EndBlockNumber)
						if err != nil {
							break
						}
						err = m.updateMsgProof(msgs, proofs, batch.ID)
						if err != nil {
							break
						}
					}
				}

			}
		}
	}()

}

func (m *MsgProofUpdater) Stop() {
	log.Info("MsgProofUpdater Stop")
}

func (m *MsgProofUpdater) initializeWithdrawTrie() error {
	var batch *orm.BridgeBatch
	firstMsg, err := m.db.GetL2SentMessageByNonce(0)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to get first l2 message: %v", err)
	}
	// no l2 message
	// 	TO DO: check if we realy dont have l2 sent message with nonce 0
	if firstMsg == nil {
		log.Info("No first l2sentmsg in db")
		return nil
	}

	// batch will never be empty, since we always have genesis batch in db
	batch, err = m.db.GetLatestBridgeBatch()
	if err != nil {
		return fmt.Errorf("failed to get latest batch: %v", err)
	}
	if batch == nil {
		return fmt.Errorf("no batch found")
	}

	var batches []*orm.BridgeBatch
	batchIndex := batch.ID
	for {
		var nonce sql.NullInt64
		// find last message nonce in before or in this batch
		nonce, err = m.db.GetLastL2MessageNonceLEHeight(batch.EndBlockNumber)
		if err != nil {
			return fmt.Errorf("failed to last l2 message nonce before %v: %v", batch.EndBlockNumber, err)
		}
		if !nonce.Valid {
			// no message before or in this batch
			break
		}

		var msg *orm.L2SentMsg
		msg, err = m.db.GetL2SentMessageByNonce(uint64(nonce.Int64))
		if err != nil {
			return fmt.Errorf("failed to l2 message with nonce %v: %v", nonce.Int64, err)
		}
		if msg.MsgProof != "" {
			// initialize withdrawTrie
			proofBytes := common.Hex2Bytes(msg.MsgProof)
			m.withdrawTrie.Initialize(uint64(nonce.Int64), common.HexToHash(msg.MsgHash), proofBytes)
			break
		}

		// append unprocessed batch
		batches = append(batches, batch)

		// iterate for next batch
		batchIndex--
		batch, err = m.db.GetBridgeBatchByIndex(batchIndex)
		if err != nil {
			return fmt.Errorf("failed to get block batch %v: %v", batchIndex, err)
		}
	}

	log.Info("Build withdraw trie with pending messages")
	for i := len(batches) - 1; i >= 0; i-- {
		batch := batches[i]
		msgs, proofs, err := m.appendL2Messages(batch.StartBlockNumber, batch.EndBlockNumber)
		if err != nil {
			return err
		}

		err = m.updateMsgProof(msgs, proofs, batch.ID)
		if err != nil {
			return err
		}
	}
	log.Info("Build withdraw trie finished")

	return nil
}

func (m *MsgProofUpdater) updateMsgProof(msgs []*orm.L2SentMsg, proofs [][]byte, batchIndex uint64) error {
	if len(msgs) > 0 {
		dbTx, err := m.db.Beginx()
		if err != nil {
			return err
		}

		for i, msg := range msgs {
			if dbTxErr := m.db.UpdateL2MessageProofInDbTx(context.Background(), dbTx, msg.MsgHash, common.Bytes2Hex(proofs[i]), batchIndex); dbTxErr != nil {
				if err := dbTx.Rollback(); err != nil {
					log.Error("dbTx.Rollback()", "err", err)
				}
				return dbTxErr
			}
		}

		if dbTxErr := dbTx.Commit(); dbTxErr != nil {
			if err := dbTx.Rollback(); err != nil {
				log.Error("dbTx.Rollback()", "err", err)
			}
			return dbTxErr
		}
	}
	return nil
}

// appendL2Messages will append all messages between firstBlock and lastBlock (both inclusive) to withdrawTrie and compute corresponding merkle proof of each message.
func (m *MsgProofUpdater) appendL2Messages(firstBlock, lastBlock uint64) ([]*orm.L2SentMsg, [][]byte, error) {
	var msgProofs [][]byte
	messages, err := m.db.GetL2SentMsgMsgHashByHeightRange(
		firstBlock,
		lastBlock,
	)
	if err != nil {
		log.Error("GetL2SentMsgMsgHashByHeightRange failed", "error", err)
		return messages, msgProofs, err
	}

	if len(messages) > 0 {
		// double check whether nonce is matched
		if messages[0].Nonce != m.withdrawTrie.NextMessageNonce {
			log.Error("L2 message nonce mismatch", "expected", messages[0].Nonce, "found", m.withdrawTrie.NextMessageNonce)
			return messages, msgProofs, fmt.Errorf("l2 message nonce mismatch, expected: %v, found: %v", messages[0].Nonce, m.withdrawTrie.NextMessageNonce)
		}

		var hashes []common.Hash
		for _, msg := range messages {
			hashes = append(hashes, common.HexToHash(msg.MsgHash))
		}
		msgProofs = m.withdrawTrie.AppendMessages(hashes)
	}
	return messages, msgProofs, nil
}
