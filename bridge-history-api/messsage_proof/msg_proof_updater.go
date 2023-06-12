package message_proof

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"bridge-history-api/db"
	"bridge-history-api/db/orm"
)

type MsgProofUpdater struct {
	client       *ethclient.Client
	startBlock   uint64
	db           db.OrmFactory
	withdrawTrie *WithdrawTrie
}

func NewMsgProofUpdater(client *ethclient.Client, startBlock uint64, db db.OrmFactory) *MsgProofUpdater {
	return &MsgProofUpdater{
		client:       client,
		db:           db,
		withdrawTrie: NewWithdrawTrie(),
	}
}

func (m *MsgProofUpdater) Start() {

}

func (m *MsgProofUpdater) Stop() {

}

func (m *MsgProofUpdater) initializeWithdrawTrie() error {
	var batch *orm.BridgeBatch
	latestBatchIndex, err := m.db.GetLatestL2SentMsgBactchIndex()
	if err != nil {
		return err
	}
	if latestBatchIndex == 0 {
		log.Info("No batch found, skip initializeWithdrawTrie, try it later")
		return nil
	}
	startBatch, err := m.db.GetBridgeBatchByBlock(m.startBlock)
	if err != nil {
		return err
	}
	for i := startBatch.Index; i < latestBatchIndex; i++ {
		batch, err = m.db.GetBridgeBatchByIndex(i)
		if err != nil {
			log.Crit("Can not initialzie withdrawtre: get batch failed: ", err)
			return err
		}
		sentMsgs, err := m.db.GetL2SentMsgMsgHashByHeightRange(batch.StartBlockNumber, batch.EndBlockNumber)
		if err != nil {
			log.Crit("Can not initialzie withdrawtre: get sent msg failed: ", err)
			return err
		}
		if len(sentMsgs) == 0 {
			continue
		}
		msgHashes := make([]common.Hash, len(sentMsgs))
		for _, msg := range sentMsgs {
			msgHashes = append(msgHashes, common.HexToHash(msg.MsgHash))
		}
		proofs := m.withdrawTrie.AppendMessages(msgHashes)

	}
	return nil
}
