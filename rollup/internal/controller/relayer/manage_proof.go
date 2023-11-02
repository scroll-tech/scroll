package relayer

import (
	"context"
	"errors"
	"scroll-tech/rollup/internal/controller/sender"
	"scroll-tech/rollup/internal/orm"
	"time"

	"github.com/scroll-tech/go-ethereum/ethclient"
)

type contractInfo struct {
	blockNumber                  uint64
	proofHashCommitEpoch         uint64
	proofCommitEpoch             uint64
	latestVerifiedBatchNumOnline uint64
}

type ProofManager struct {
	ctx            context.Context
	exit           context.CancelFunc
	cfg            Config
	batchOrm       *orm.Batch
	client  		*ethclient.Client

	commitSender   *sender.Sender
	finalizeSender *sender.Sender

	finalProofCh       chan<- finalProofMsg
	proofHashCh        chan proofHash
	sendFailProofMsgCh <-chan sendFailProofMsg
	proofSender        SendProofServiceServer
	batchNumber        int
}


func (pm *ProofManager) tryFetchProofToSend(ctx context.Context) {
	var lastVerifiedBatchNum uint64
	var nextBatchNum uint64
	tick := time.NewTicker(time.Second * 1)

	nextBatchNum = uint64(pm.batchNumber)
	contractInfo := contractInfo{}

}

func (pm *ProofManager) updateContractInfo(info *contractInfo) error {
	if info == nil {
		return errors.New("Input info is nill")
	}
	curBlockNumber, err := pm.client.BlockNumber(pm.ctx)
	if err != nil {
		return err
	}
	proofHashCommitEpoch, err := pm.client.GetProofHashCommitEpoch()
	if err != nil {
		return err
	}

}