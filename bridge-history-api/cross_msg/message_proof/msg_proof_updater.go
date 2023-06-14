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
	m.initialize(m.ctx)
	go func() {
		tick := time.NewTicker(10 * time.Second)
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
	}()

}

func (m *MsgProofUpdater) Stop() {
	log.Info("MsgProofUpdater Stop")
}

func (m *MsgProofUpdater) initialize(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			err := m.initializeWithdrawTrie()
			if err != nil {
				log.Error("can not initialize withdraw trie", "err", err)
				continue
			}
			return
		}
	}
}

func (m *MsgProofUpdater) initializeWithdrawTrie() error {
	var batches []*orm.BridgeBatch
	firstMsg, err := m.db.GetL2SentMessageByNonce(0)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to get first l2 message: %v", err)
	}
	//  no l2 message in db
	if firstMsg == nil {
		log.Info("No first l2sentmsg in db")
		return nil
	}

	// batch will never be empty, since we always have genesis batch in db
	endBatch, err := m.db.GetLatestBridgeBatch()
	if err != nil {
		return fmt.Errorf("failed to get latest batch: %v", err)
	}
	if endBatch == nil {
		return fmt.Errorf("no batch found")
	}
	var startIndex uint64
	startBatch, err := m.db.GetLatestBridgeBatchWithProof()
	if err != nil {
		return fmt.Errorf("failed to get latest batch with proof: %v", err)
	}
	if startBatch == nil {
		startBatch, err = m.db.GetBridgeBatchByIndex(1)
		if err != nil {
			return fmt.Errorf("failed to get batch by index 1: %v", err)
		}
	} else {
		startIndex = startBatch.ID
	}

	msg, err := m.db.GetL2SentMsgMsgHashByHeightRange(startBatch.StartBlockNumber, startBatch.EndBlockNumber)
	if err != nil {
		return fmt.Errorf("failed to get latest l2 sent msg le height %d: %v", startBatch.EndBlockNumber, err)
	}
	last := msg[len(msg)-1]
	if last.MsgProof != "" {
		// already have proof
		// initialize withdrawTrie
		proofBytes := common.Hex2Bytes(last.MsgProof)
		m.withdrawTrie.Initialize(last.Nonce, common.HexToHash(last.MsgHash), proofBytes)
	} else {
		batches = append(batches, startBatch)
	}

	for i := startIndex + 1; i <= endBatch.ID; i++ {
		iterBatch, err := m.db.GetBridgeBatchByIndex(i)
		if err != nil {
			return fmt.Errorf("failed to get batch by index %d: %v", i, err)
		}
		batches = append(batches, iterBatch)

	}
	log.Info("Build withdraw trie with pending messages")
	for _, b := range batches {
		msgs, proofs, err := m.appendL2Messages(b.StartBlockNumber, b.EndBlockNumber)
		if err != nil {
			return err
		}
		err = m.updateMsgProof(msgs, proofs, b.ID)
		if err != nil {
			return err
		}
	}
	log.Info("Build withdraw trie finished")
	return nil
}

func (m *MsgProofUpdater) updateMsgProof(msgs []*orm.L2SentMsg, proofs [][]byte, batchIndex uint64) error {
	if len(msgs) == 0 {
		return nil
	}

	dbTx, err := m.db.Beginx()
	if err != nil {
		return err
	}

	dbTxErr := m.db.UpdateBridgeBatchStatusDBTx(dbTx, batchIndex, orm.BatchWithProof)
	if dbTxErr != nil {
		if err := dbTx.Rollback(); err != nil {
			log.Error("dbTx.Rollback()", "err", err)
		}
		return dbTxErr
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

	return nil
}

// appendL2Messages will append all messages between firstBlock and lastBlock (both inclusive) to withdrawTrie and compute corresponding merkle proof of each message.
func (m *MsgProofUpdater) appendL2Messages(firstBlock, lastBlock uint64) ([]*orm.L2SentMsg, [][]byte, error) {
	var msgProofs [][]byte
	messages, err := m.db.GetL2SentMsgMsgHashByHeightRange(firstBlock, lastBlock)
	if err != nil {
		log.Error("GetL2SentMsgMsgHashByHeightRange failed", "error", err)
		return messages, msgProofs, err
	}

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

	return messages, msgProofs, nil
}
