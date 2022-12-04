package orm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/scroll-tech/go-ethereum/log"
)

type layer2MessageOrm struct {
	db *sqlx.DB
}

var _ L2MessageOrm = (*layer2MessageOrm)(nil)

// NewL2MessageOrm create an L2MessageOrm instance
func NewL2MessageOrm(db *sqlx.DB) L2MessageOrm {
	return &layer2MessageOrm{db: db}
}

// GetL2MessageByNonce fetch message by nonce
func (m *layer2MessageOrm) GetL2MessageByNonce(nonce uint64) (*L2Message, error) {
	msg := L2Message{}

	row := m.db.QueryRowx(`SELECT nonce, msg_hash, height, sender, target, value, fee, gas_limit, deadline, calldata, layer2_hash, status FROM l2_message WHERE nonce = $1`, nonce)
	if err := row.StructScan(&msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// GetL2MessageByMsgHash fetch message by message hash
func (m *layer2MessageOrm) GetL2MessageByMsgHash(msgHash string) (*L2Message, error) {
	msg := L2Message{}

	row := m.db.QueryRowx(`SELECT nonce, msg_hash, height, sender, target, value, fee, gas_limit, deadline, calldata, layer2_hash, status FROM l2_message WHERE msg_hash = $1`, msgHash)
	if err := row.StructScan(&msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// GetMessageProofByNonce fetch message by nonce
func (m *layer2MessageOrm) GetMessageProofByNonce(nonce uint64) (string, error) {
	row := m.db.QueryRow(`SELECT proof FROM l2_message WHERE nonce = $1`, nonce)
	var proof sql.NullString
	if err := row.Scan(&proof); err != nil {
		return "", err
	}
	if proof.Valid {
		return proof.String, nil
	}
	return "", nil
}

// MessageProofExist fetch message by nonce
func (m *layer2MessageOrm) MessageProofExist(nonce uint64) (bool, error) {
	err := m.db.QueryRow(`SELECT nonce FROM l2_message WHERE nonce = $1 and proof IS NOT NULL`, nonce).Scan(&nonce)
	if err != nil {
		if err != sql.ErrNoRows {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

// GetL2ProcessedNonce fetch latest processed message nonce
func (m *layer2MessageOrm) GetL2ProcessedNonce() (int64, error) {
	row := m.db.QueryRow(`SELECT MAX(nonce) FROM l2_message WHERE status = $1;`, MsgConfirmed)

	// no row means no message
	// since nonce starts with 0, return -1 as the processed nonce
	var nonce sql.NullInt64
	if err := row.Scan(&nonce); err != nil {
		if err == sql.ErrNoRows || !nonce.Valid {
			return -1, nil
		}
		return 0, err
	}
	if nonce.Valid {
		return nonce.Int64, nil
	}
	return -1, nil
}

// GetL2MessagesByStatus fetch list of messages given msg status
func (m *layer2MessageOrm) GetL2MessagesByStatus(status MsgStatus) ([]*L2Message, error) {
	rows, err := m.db.Queryx(`SELECT nonce, msg_hash, height, sender, target, value, fee, gas_limit, deadline, calldata, layer2_hash FROM l2_message WHERE status = $1 ORDER BY nonce ASC;`, status)
	if err != nil {
		return nil, err
	}

	var msgs []*L2Message
	for rows.Next() {
		msg := &L2Message{}
		if err = rows.StructScan(&msg); err != nil {
			break
		}
		msgs = append(msgs, msg)
	}
	if len(msgs) == 0 || errors.Is(err, sql.ErrNoRows) {
		// log.Warn("no unprocessed layer2 messages in db", "err", err)
	} else if err != nil {
		return nil, err
	}

	return msgs, rows.Close()
}

// SaveL2Messages batch save a list of layer2 messages
func (m *layer2MessageOrm) SaveL2Messages(ctx context.Context, messages []*L2Message) error {
	messageMaps := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		messageMaps[i] = map[string]interface{}{
			"nonce":       msg.Nonce,
			"msg_hash":    msg.MsgHash,
			"height":      msg.Height,
			"sender":      msg.Sender,
			"target":      msg.Target,
			"value":       msg.Value,
			"fee":         msg.Fee,
			"gas_limit":   msg.GasLimit,
			"deadline":    msg.Deadline,
			"calldata":    msg.Calldata,
			"layer2_hash": msg.Layer2Hash,
		}
	}

	_, err := m.db.NamedExec(`INSERT INTO public.l2_message (nonce, msg_hash, height, sender, target, value, fee, gas_limit, deadline, calldata, layer2_hash) VALUES (:nonce, :msg_hash, :height, :sender, :target, :value, :fee, :gas_limit, :deadline, :calldata, :layer2_hash);`, messageMaps)
	if err != nil {
		nonces := make([]uint64, 0, len(messages))
		heights := make([]uint64, 0, len(messages))
		for _, msg := range messages {
			nonces = append(nonces, msg.Nonce)
			heights = append(heights, msg.Height)
		}
		log.Error("failed to insert layer2Messages", "nonces", nonces, "heights", heights, "err", err)
	}
	return err
}

// UpdateLayer1Hash update corresponding layer1 hash, given message hash
func (m *layer2MessageOrm) UpdateLayer1Hash(ctx context.Context, msgHash, layer1Hash string) error {
	if _, err := m.db.ExecContext(ctx, m.db.Rebind("update l2_message set layer1_hash = ? where msg_hash = ?;"), layer1Hash, msgHash); err != nil {
		return err
	}

	return nil
}

// UpdateMessageProof update corresponding message proof, given message nonce
func (m *layer2MessageOrm) UpdateMessageProof(ctx context.Context, nonce uint64, proof string) error {
	if _, err := m.db.ExecContext(ctx, m.db.Rebind("update l2_message set proof = ? where nonce = ?;"), proof, nonce); err != nil {
		return err
	}

	return nil
}

// UpdateLayer2Status updates message stauts, given message hash
func (m *layer2MessageOrm) UpdateLayer2Status(ctx context.Context, msgHash string, status MsgStatus) error {
	if _, err := m.db.ExecContext(ctx, m.db.Rebind("update l2_message set status = ? where msg_hash = ?;"), status, msgHash); err != nil {
		return err
	}

	return nil
}

// UpdateLayer2StatusAndLayer1Hash updates message stauts and layer1 transaction hash, given message hash
func (m *layer2MessageOrm) UpdateLayer2StatusAndLayer1Hash(ctx context.Context, msgHash, layer1Hash string, status MsgStatus) error {
	if _, err := m.db.ExecContext(ctx, m.db.Rebind("update l2_message set status = ?, layer1_hash = ? where msg_hash = ?;"), status, layer1Hash, msgHash); err != nil {
		return err
	}

	return nil
}

// GetLayer2LatestWatchedHeight returns latest height stored in the table
func (m *layer2MessageOrm) GetLayer2LatestWatchedHeight() (int64, error) {
	// @note It's not correct, since we may don't have message in some blocks.
	// But it will only be called at start, some redundancy is acceptable.
	row := m.db.QueryRow("SELECT COALESCE(MAX(height), -1) FROM l2_message;")

	var height int64
	if err := row.Scan(&height); err != nil {
		return -1, err
	}
	if height < 0 {
		return -1, fmt.Errorf("could not get height due to database return negative")
	}
	return height, nil
}
