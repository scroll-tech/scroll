package l1

import (
	"context"
	"errors"
	"math/big"
	"time"

	// not sure if this will make problems when relay with l1geth

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/ethclient/gethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"golang.org/x/sync/errgroup"

	"scroll-tech/database"
	"scroll-tech/database/orm"

	bridge_abi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/config"
	"scroll-tech/bridge/sender"
	"scroll-tech/bridge/utils"
)

// Layer1Relayer is responsible for
//  1. fetch pending L1Message from db
//  2. relay pending message to layer 2 node
//
// Actions are triggered by new head from layer 1 geth node.
// @todo It's better to be triggered by watcher.
type Layer1Relayer struct {
	ctx context.Context

	gethClient *gethclient.Client
	ethClient  *ethclient.Client

	// sender and channel used for relay message
	relaySender         *sender.Sender
	relayConfirmationCh <-chan *sender.Confirmation

	// sender and channel used for import blocks
	importSender         *sender.Sender
	importConfirmationCh <-chan *sender.Confirmation

	l1MessageQueueAddress common.Address

	db  database.OrmFactory
	cfg *config.RelayerConfig

	l2MessengerABI      *abi.ABI
	l1BlockContainerABI *abi.ABI

	stopCh chan struct{}
}

// NewLayer1Relayer will return a new instance of Layer1RelayerClient
func NewLayer1Relayer(ctx context.Context, gethClient *gethclient.Client, ethClient *ethclient.Client, l1ConfirmNum int64, l1MessageQueueAddress common.Address, db database.OrmFactory, cfg *config.RelayerConfig) (*Layer1Relayer, error) {
	relaySender, err := sender.NewSender(ctx, cfg.SenderConfig, cfg.MessageSenderPrivateKeys)
	if err != nil {
		log.Error("new relayer sender failed", "err", err)
		return nil, err
	}

	if len(cfg.RollupSenderPrivateKeys) != 1 {
		return nil, errors.New("more than 1 private key for importing L1 block")
	}
	importSender, err := sender.NewSender(ctx, cfg.SenderConfig, cfg.RollupSenderPrivateKeys)
	if err != nil {
		addr := crypto.PubkeyToAddress(cfg.MessageSenderPrivateKeys[0].PublicKey)
		log.Error("new sender failed", "main address", addr.String(), "err", err)
		return nil, err
	}

	return &Layer1Relayer{
		ctx: ctx,

		gethClient: gethClient,
		ethClient:  ethClient,

		relaySender:         relaySender,
		relayConfirmationCh: relaySender.ConfirmChan(),

		importSender:         importSender,
		importConfirmationCh: importSender.ConfirmChan(),

		l1MessageQueueAddress: l1MessageQueueAddress,

		db:  db,
		cfg: cfg,

		l2MessengerABI:      bridge_abi.L2MessengerABI,
		l1BlockContainerABI: bridge_abi.L1BlockContainerABI,

		stopCh: make(chan struct{}),
	}, nil
}

// ProcessSavedEvents relays saved un-processed cross-domain transactions to desired blockchain
func (r *Layer1Relayer) ProcessSavedEvents() {
	block, err := r.db.GetLatestImportedL1Block()
	if err != nil {
		log.Error("GetLatestImportedL1Block failed", "err", err)
		return
	}

	// msgs are sorted by nonce in increasing order
	msgs, err := r.db.GetL1MessagesByStatusUpToProofHeight(orm.MsgPending, block.Number)
	if err != nil {
		log.Error("Failed to fetch unprocessed L1 messages", "err", err)
		return
	}

	// process messages in batches
	batch_size := r.relaySender.NumberOfAccounts()
	for from := 0; from < len(msgs); from += batch_size {
		to := from + batch_size
		if to > len(msgs) {
			to = len(msgs)
		}

		var g errgroup.Group

		for i := from; i < to; i++ {
			msg := msgs[i]

			g.Go(func() error {
				return r.processSavedEvent(msg, block)
			})
		}

		if err := g.Wait(); err != nil {
			if !errors.Is(err, sender.ErrNoAvailableAccount) {
				log.Error("failed to process l1 saved event", "err", err)
			}
			return
		}
	}
}

func (r *Layer1Relayer) processSavedEvent(msg *orm.L1Message, block *orm.L1BlockInfo) error {
	// @todo add support to relay multiple messages
	from := common.HexToAddress(msg.Sender)
	target := common.HexToAddress(msg.Target)
	value, ok := big.NewInt(0).SetString(msg.Value, 10)
	if !ok {
		// @todo maybe panic?
		log.Error("Failed to parse message value", "nonce", msg.Nonce, "height", msg.Height, "value", msg.Value)
		// TODO: need to skip this message by changing its status to MsgError
	}

	if len(msg.MessageProof) == 0 {
		// empty proof, we need to fetch storage proof from client
		hash := common.HexToHash(msg.MsgHash)
		proofs, err := utils.GetL1MessageProof(r.gethClient, r.l1MessageQueueAddress, []common.Hash{hash}, block.Number)
		if err != nil {
			log.Error("Failed to GetL1MessageProof", "msg.hash", msg.MsgHash, "height", block.Number, "err", err)
			return err
		}
		msg.MessageProof = common.Bytes2Hex(proofs[0])
		msg.ProofHeight = block.Number
	}

	blocks, err := r.db.GetL1BlockInfos(map[string]interface{}{
		"number": msg.ProofHeight,
	})
	if err != nil {
		log.Error("Failed to GetL1BlockInfos from db", "proof_height", msg.ProofHeight, "err", err)
		return err
	}
	if len(blocks) != 1 {
		log.Error("Block not exist", "height", msg.ProofHeight)
		return errors.New("block not exist")
	}

	proof := bridge_abi.IL2ScrollMessengerL1MessageProof{
		BlockHash:      common.HexToHash(blocks[0].Hash),
		StateRootProof: common.Hex2Bytes(msg.MessageProof),
	}

	fee, _ := big.NewInt(0).SetString(msg.Fee, 10)
	deadline := big.NewInt(int64(msg.Deadline))
	msgNonce := big.NewInt(int64(msg.Nonce))
	calldata := common.Hex2Bytes(msg.Calldata)
	data, err := r.l2MessengerABI.Pack("relayMessageWithProof", from, target, value, fee, deadline, msgNonce, calldata, proof)
	if err != nil {
		log.Error("Failed to pack relayMessageWithProof", "msg.nonce", msg.Nonce, "msg.height", msg.Height, "err", err)
		// TODO: need to skip this message by changing its status to MsgError
		return err
	}

	hash, err := r.relaySender.SendTransaction(msg.MsgHash, &r.cfg.MessengerContractAddress, big.NewInt(0), data)
	if err != nil {
		return err
	}
	log.Info("relayMessage to layer2", "msg hash", msg.MsgHash, "tx hash", hash)

	err = r.db.UpdateLayer1StatusAndLayer2Hash(r.ctx, msg.MsgHash, orm.MsgSubmitted, hash.String())
	if err != nil {
		log.Error("UpdateLayer1StatusAndLayer2Hash failed", "msg.msgHash", msg.MsgHash, "msg.height", msg.Height, "err", err)
	}
	return err
}

// ProcessPendingBlocks imports failed/pending block headers to layer2
func (r *Layer1Relayer) ProcessPendingBlocks() {
	// handle failed block first since we need to import sequentially
	failedBlocks, err := r.db.GetL1BlockInfos(map[string]interface{}{
		"block_status": orm.L1BlockFailed,
	})
	if err != nil {
		log.Error("Failed to fetch failed L1 Blocks from db", "err", err)
		return
	}
	for _, block := range failedBlocks {
		if err = r.importBlock(block); err != nil {
			if !errors.Is(err, sender.ErrNoAvailableAccount) {
				log.Error("failed to retry failed L1 block", "err", err)
			}
			return
		}
	}

	// If there are failed blocks, we don't handle pending blocks. This is
	// because if there are `importing`` blocks after `failed`` blocks, the
	// `importing` block will fail eventually. If we send `pending` blocks
	// immediately, they will also fail eventually.
	if len(failedBlocks) > 0 {
		return
	}

	// handle pending blocks
	pendingBlocks, err := r.db.GetL1BlockInfos(map[string]interface{}{
		"block_status": orm.L1BlockPending,
	})
	if err != nil {
		log.Error("Failed to fetch pending L1 Blocks from db", "err", err)
		return
	}
	for _, block := range pendingBlocks {
		if err = r.importBlock(block); err != nil {
			if !errors.Is(err, sender.ErrNoAvailableAccount) {
				log.Error("failed to import pending L1 block", "err", err)
			}
			return
		}
	}
}

func (r *Layer1Relayer) importBlock(block *orm.L1BlockInfo) error {
	data, err := r.l1BlockContainerABI.Pack("importBlockHeader", common.HexToHash(block.Hash), common.Hex2Bytes(block.HeaderRLP), make([]byte, 0))
	if err != nil {
		return err
	}

	hash, err := r.importSender.SendTransaction(block.Hash, &r.cfg.ContrainerContractAddress, big.NewInt(0), data)
	if err != nil {
		return err
	}
	log.Info("import block to layer2", "height", block.Number, "hash", block.Hash, "tx hash", hash)

	err = r.db.UpdateL1BlockStatusAndImportTxHash(r.ctx, block.Hash, orm.L1BlockImporting, hash.String())
	if err != nil {
		log.Error("UpdateL1BlockStatusAndImportTxHash failed", "height", block.Number, "hash", block.Hash, "err", err)
	}
	return err
}

// Start the relayer process
func (r *Layer1Relayer) Start() {
	go func() {
		// trigger by timer
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				r.ProcessPendingBlocks()
				r.ProcessSavedEvents()
			case cfm := <-r.relayConfirmationCh:
				if !cfm.IsSuccessful {
					// @todo handle db error
					err := r.db.UpdateLayer1StatusAndLayer2Hash(r.ctx, cfm.ID, orm.MsgFailed, cfm.TxHash.String())
					if err != nil {
						log.Warn("UpdateLayer1StatusAndLayer2Hash failed", "err", err)
					}
					log.Warn("relay transaction confirmed but failed in layer2", "confirmation", cfm)
				} else {
					// @todo handle db error
					err := r.db.UpdateLayer1StatusAndLayer2Hash(r.ctx, cfm.ID, orm.MsgConfirmed, cfm.TxHash.String())
					if err != nil {
						log.Warn("UpdateLayer1StatusAndLayer2Hash failed", "err", err)
					}
					log.Info("relay transaction confirmed in layer2", "confirmation", cfm)
				}
			case cfm := <-r.importConfirmationCh:
				if !cfm.IsSuccessful {
					// @todo handle db error
					err := r.db.UpdateL1BlockStatusAndImportTxHash(r.ctx, cfm.ID, orm.L1BlockFailed, cfm.TxHash.String())
					if err != nil {
						log.Warn("UpdateL1BlockStatusAndImportTxHash failed", "err", err)
					}
					log.Warn("import transaction confirmed but failed in layer2", "confirmation", cfm)
				} else {
					// @todo handle db error
					err := r.db.UpdateL1BlockStatusAndImportTxHash(r.ctx, cfm.ID, orm.L1BlockImported, cfm.TxHash.String())
					if err != nil {
						log.Warn("UpdateL1BlockStatusAndImportTxHash failed", "err", err)
					}
					log.Info("import transaction confirmed in layer2", "confirmation", cfm)
				}
			case <-r.stopCh:
				return
			}
		}
	}()
}

// Stop the relayer module, for a graceful shutdown.
func (r *Layer1Relayer) Stop() {
	close(r.stopCh)
}
