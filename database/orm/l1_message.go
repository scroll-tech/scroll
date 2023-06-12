package orm

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/types"
)

type l1MessageOrm struct {
	db *sqlx.DB
}

var _ L1MessageOrm = (*l1MessageOrm)(nil)

// NewL1MessageOrm create an L1MessageOrm instance
func NewL1MessageOrm(db *sqlx.DB) L1MessageOrm {
	return &l1MessageOrm{db: db}
}

// GetL1MessageByMsgHash fetch message by queue_index
func (m *l1MessageOrm) GetL1MessageByMsgHash(msgHash string) (*types.L1Message, error) {
	msg := types.L1Message{}

	row := m.db.QueryRowx(`SELECT queue_index, msg_hash, height, sender, target, value, gas_limit, calldata, layer1_hash, status FROM l1_message WHERE msg_hash = $1`, msgHash)

	if err := row.StructScan(&msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// GetL1MessageByQueueIndex fetch message by queue_index
func (m *l1MessageOrm) GetL1MessageByQueueIndex(queueIndex uint64) (*types.L1Message, error) {
	msg := types.L1Message{}

	row := m.db.QueryRowx(`SELECT queue_index, msg_hash, height, sender, target, value, calldata, layer1_hash, status FROM l1_message WHERE queue_index = $1`, queueIndex)

	if err := row.StructScan(&msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// GetL1MessagesByStatus fetch list of unprocessed messages given msg status
func (m *l1MessageOrm) GetL1MessagesByStatus(status types.MsgStatus, limit uint64) ([]*types.L1Message, error) {
	rows, err := m.db.Queryx(`SELECT queue_index, msg_hash, height, sender, target, value, calldata, layer1_hash, status FROM l1_message WHERE status = $1 ORDER BY queue_index ASC LIMIT $2;`, status, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var msgs []*types.L1Message
	for rows.Next() {
		msg := &types.L1Message{}
		if err = rows.StructScan(&msg); err != nil {
			break
		}
		msgs = append(msgs, msg)
	}
	if len(msgs) == 0 || errors.Is(err, sql.ErrNoRows) {
		// log.Warn("no unprocessed layer1 messages in db", "err", err)
	} else if err != nil {
		return nil, err
	}

	return msgs, nil
}

// GetL1ProcessedQueueIndex fetch latest processed message queue_index
func (m *l1MessageOrm) GetL1ProcessedQueueIndex() (int64, error) {
	row := m.db.QueryRow(`SELECT MAX(queue_index) FROM l1_message WHERE status = $1;`, types.MsgConfirmed)

	var queueIndex sql.NullInt64
	if err := row.Scan(&queueIndex); err != nil {
		if err == sql.ErrNoRows || !queueIndex.Valid {
			// no row means no message
			// since queueIndex starts with 0, return -1 as the processed queueIndex
			return -1, nil
		}
		return 0, err
	}
	if queueIndex.Valid {
		return queueIndex.Int64, nil
	}
	return -1, nil
}

// SaveL1Messages batch save a list of layer1 messages
func (m *l1MessageOrm) SaveL1Messages(ctx context.Context, messages []*types.L1Message) error {
	if len(messages) == 0 {
		return nil
	}

	messageMaps := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {

		messageMaps[i] = map[string]interface{}{
			"queue_index": msg.QueueIndex,
			"msg_hash":    msg.MsgHash,
			"height":      msg.Height,
			"sender":      msg.Sender,
			"target":      msg.Target,
			"value":       msg.Value,
			"gas_limit":   msg.GasLimit,
			"calldata":    msg.Calldata,
			"layer1_hash": msg.Layer1Hash,
		}
	}
	_, err := m.db.NamedExec(`INSERT INTO public.l1_message (queue_index, msg_hash, height, sender, target, value, gas_limit, calldata, layer1_hash) VALUES (:queue_index, :msg_hash, :height, :sender, :target, :value, :gas_limit, :calldata, :layer1_hash);`, messageMaps)
	if err != nil {
		queueIndices := make([]uint64, 0, len(messages))
		heights := make([]uint64, 0, len(messages))
		for _, msg := range messages {
			queueIndices = append(queueIndices, msg.QueueIndex)
			heights = append(heights, msg.Height)
		}
		log.Error("failed to insert l1Messages", "queueIndices", queueIndices, "heights", heights, "err", err)
	}
	return err
}

// UpdateLayer2Hash update corresponding layer2 hash, given message hash
func (m *l1MessageOrm) UpdateLayer2Hash(ctx context.Context, msgHash, layer2Hash string) error {
	if _, err := m.db.ExecContext(ctx, m.db.Rebind("update l1_message set layer2_hash = ? where msg_hash = ?;"), layer2Hash, msgHash); err != nil {
		return err
	}

	return nil
}

// UpdateLayer1Status updates message stauts, given message hash
func (m *l1MessageOrm) UpdateLayer1Status(ctx context.Context, msgHash string, status types.MsgStatus) error {
	if _, err := m.db.ExecContext(ctx, m.db.Rebind("update l1_message set status = ? where msg_hash = ?;"), status, msgHash); err != nil {
		return err
	}

	return nil
}

// UpdateLayer1StatusAndLayer2Hash updates message status and layer2 transaction hash, given message hash
func (m *l1MessageOrm) UpdateLayer1StatusAndLayer2Hash(ctx context.Context, msgHash string, status types.MsgStatus, layer2Hash string) error {
	if _, err := m.db.ExecContext(ctx, m.db.Rebind("update l1_message set status = ?, layer2_hash = ? where msg_hash = ?;"), status, layer2Hash, msgHash); err != nil {
		return err
	}

	return nil
}

// GetLayer1LatestWatchedHeight returns latest height stored in the table
func (m *l1MessageOrm) GetLayer1LatestWatchedHeight() (int64, error) {
	// @note It's not correct, since we may don't have message in some blocks.
	// But it will only be called at start, some redundancy is acceptable.
	row := m.db.QueryRow("SELECT MAX(height) FROM l1_message;")

	var height sql.NullInt64
	if err := row.Scan(&height); err != nil {
		if err == sql.ErrNoRows || !height.Valid {
			return -1, nil
		}
		return 0, err
	}
	if height.Valid {
		return height.Int64, nil
	}
	return -1, nil
}

func (m *l1MessageOrm) GetRelayL1MessageTxHash(queueIndex uint64) (sql.NullString, error) {
	row := m.db.QueryRow(`SELECT layer2_hash FROM l1_message WHERE queue_index = $1`, queueIndex)
	var hash sql.NullString
	if err := row.Scan(&hash); err != nil {
		return sql.NullString{}, err
	}
	return hash, nil
}
