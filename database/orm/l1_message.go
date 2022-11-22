package orm

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/scroll-tech/go-ethereum/log"
)

type l1MessageOrm struct {
	db *sqlx.DB
}

var _ L1MessageOrm = (*l1MessageOrm)(nil)

// NewL1MessageOrm create an L1MessageOrm instance
func NewL1MessageOrm(db *sqlx.DB) L1MessageOrm {
	return &l1MessageOrm{db: db}
}

// GetL1MessageByLayer1Hash fetch message by nonce
func (m *l1MessageOrm) GetL1MessageByLayer1Hash(layer1Hash string) (*L1Message, error) {
	msg := L1Message{}

	row := m.db.QueryRow(`SELECT nonce, height, sender, target, value, fee, gas_limit, deadline, calldata, layer1_hash, status FROM l1_message WHERE layer1_hash = $1`, layer1Hash)

	if err := row.Scan(&msg.Nonce, &msg.Height, &msg.Sender, &msg.Target, &msg.Value, &msg.Fee, &msg.GasLimit, &msg.Deadline, &msg.Calldata, &msg.Layer1Hash, &msg.Status); err != nil {
		return nil, err
	}
	return &msg, nil
}

// GetL1MessageByNonce fetch message by nonce
func (m *l1MessageOrm) GetL1MessageByNonce(nonce uint64) (*L1Message, error) {
	msg := L1Message{}

	row := m.db.QueryRow(`SELECT nonce, height, sender, target, value, fee, gas_limit, deadline, calldata, layer1_hash, status FROM l1_message WHERE nonce = $1`, nonce)

	if err := row.Scan(&msg.Nonce, &msg.Height, &msg.Sender, &msg.Target, &msg.Value, &msg.Fee, &msg.GasLimit, &msg.Deadline, &msg.Calldata, &msg.Layer1Hash, &msg.Status); err != nil {
		return nil, err
	}
	return &msg, nil
}

// GetL1UnprocessedMessages fetch list of unprocessed messages
func (m *l1MessageOrm) GetL1UnprocessedMessages() ([]*L1Message, error) {
	rows, err := m.db.Queryx(`SELECT nonce, height, sender, target, value, fee, gas_limit, deadline, calldata, layer1_hash, status FROM l1_message WHERE status = $1 ORDER BY nonce ASC;`, MsgPending)
	if err != nil {
		return nil, err
	}

	var msgs []*L1Message
	for rows.Next() {
		msg := &L1Message{}
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

	return msgs, rows.Close()
}

// GetL1ProcessedNonce fetch latest processed message nonce
func (m *l1MessageOrm) GetL1ProcessedNonce() (int64, error) {
	row := m.db.QueryRow(`SELECT MAX(nonce) FROM l1_message WHERE status = $1;`, MsgConfirmed)

	var nonce int64
	err := row.Scan(&nonce)
	if err != nil {
		if err == sql.ErrNoRows {
			// no row means no message
			// since nonce starts with 0, return -1 as the processed nonce
			return -1, nil
		}
		return 0, err
	}
	return nonce, nil
}

// SaveL1Messages batch save a list of layer1 messages
func (m *l1MessageOrm) SaveL1Messages(ctx context.Context, messages []*L1Message) error {
	messageMaps := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		messageMaps[i] = map[string]interface{}{
			"nonce":       msg.Nonce,
			"height":      msg.Height,
			"sender":      msg.Sender,
			"target":      msg.Target,
			"value":       msg.Value,
			"fee":         msg.Fee,
			"gas_limit":   msg.GasLimit,
			"deadline":    msg.Deadline,
			"calldata":    msg.Calldata,
			"layer1_hash": msg.Layer1Hash,
		}
	}
	_, err := m.db.NamedExec(`INSERT INTO public.l1_message (nonce, height, sender, target, value, fee, gas_limit, deadline, calldata, layer1_hash) VALUES (:nonce, :height, :sender, :target, :value, :fee, :gas_limit, :deadline, :calldata, :layer1_hash);`, messageMaps)
	if err != nil {
		nonces := make([]uint64, 0, len(messages))
		heights := make([]uint64, 0, len(messages))
		for _, msg := range messages {
			nonces = append(nonces, msg.Nonce)
			heights = append(heights, msg.Height)
		}
		log.Error("failed to insert l1Messages", "nonces", nonces, "heights", heights, "err", err)
	}
	return err
}

// UpdateLayer2Hash update corresponding layer2 hash given message nonce
func (m *l1MessageOrm) UpdateLayer2Hash(ctx context.Context, layer1Hash string, layer2Hash string) error {
	if _, err := m.db.ExecContext(ctx, m.db.Rebind("update l1_message set layer2_hash = ? where layer1_hash = ?;"), layer2Hash, layer1Hash); err != nil {
		return err
	}

	return nil
}

// UpdateLayer1Status updates message stauts
func (m *l1MessageOrm) UpdateLayer1Status(ctx context.Context, layer1Hash string, status MsgStatus) error {
	if _, err := m.db.ExecContext(ctx, m.db.Rebind("update l1_message set status = ? where layer1_hash = ?;"), status, layer1Hash); err != nil {
		return err
	}

	return nil
}

// UpdateLayer1StatusAndLayer2Hash updates message status and layer2 transaction hash
func (m *l1MessageOrm) UpdateLayer1StatusAndLayer2Hash(ctx context.Context, layer1Hash, layer2Hash string, status MsgStatus) error {
	if _, err := m.db.ExecContext(ctx, m.db.Rebind("update l1_message set status = ?, layer2_hash = ? where layer1_hash = ?;"), status, layer2Hash, layer1Hash); err != nil {
		return err
	}

	return nil
}

// GetLayer1LatestWatchedHeight returns latest height stored in the table
func (m *l1MessageOrm) GetLayer1LatestWatchedHeight() (int64, error) {
	// @note It's not correct, since we may don't have message in some blocks.
	// But it will only be called at start, some redundancy is acceptable.
	row := m.db.QueryRow("SELECT MAX(height) FROM l1_message;")

	var height int64
	if err := row.Scan(&height); err != nil {
		if err == sql.ErrNoRows {
			return -1, nil
		}
		return 0, err
	}
	return height, nil
}
