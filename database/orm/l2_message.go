package orm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"

	"github.com/islishude/bigint"
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

	row := m.db.QueryRow(`SELECT nonce, height, sender, target, value, fee, gas_limit, deadline, calldata, layer2_hash, status FROM l2_message WHERE nonce = $1`, nonce)
	if err := row.Scan(&msg.Nonce, &msg.Height, &msg.Sender, &msg.Target, &msg.Value, &msg.Fee, &msg.GasLimit, &msg.Deadline, &msg.Calldata, &msg.Layer2Hash, &msg.Status); err != nil {
		return nil, err
	}
	return &msg, nil
}

// GetL2MessageByLayer2Hash fetch message by layer2Hash
func (m *layer2MessageOrm) GetL2MessageByLayer2Hash(layer2Hash string) (*L2Message, error) {
	msg := L2Message{}

	row := m.db.QueryRow(`SELECT nonce, height, sender, target, value, fee, gas_limit, deadline, calldata, layer2_hash, status FROM l2_message WHERE layer2_hash = $1`, layer2Hash)
	if err := row.Scan(&msg.Nonce, &msg.Height, &msg.Sender, &msg.Target, &msg.Value, &msg.Fee, &msg.GasLimit, &msg.Deadline, &msg.Calldata, &msg.Layer2Hash, &msg.Status); err != nil {
		return nil, err
	}
	return &msg, nil
}

// GetMessageProofByLayer2Hash fetch message proof by layer2Hash
func (m *layer2MessageOrm) GetMessageProofByLayer2Hash(layer2Hash string) (string, error) {
	row := m.db.QueryRow(`SELECT proof FROM l2_message WHERE layer2_hash = $1`, layer2Hash)
	var proof string
	if err := row.Scan(&proof); err != nil {
		return "", err
	}
	return proof, nil
}

// MessageProofExistByLayer2Hash fetch message by layer2Hash
func (m *layer2MessageOrm) MessageProofExistByLayer2Hash(layer2Hash string) (bool, error) {
	err := m.db.QueryRow(`SELECT layer2_hash FROM l2_message WHERE layer2_hash = $1 and proof IS NOT NULL`, layer2Hash).Scan(&layer2Hash)
	if err != nil {
		if err != sql.ErrNoRows {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

// GetMessageProofByNonce fetch message by nonce
func (m *layer2MessageOrm) GetMessageProofByNonce(nonce uint64) (string, error) {
	row := m.db.QueryRow(`SELECT proof FROM l2_message WHERE nonce = $1`, nonce)
	var proof string
	if err := row.Scan(&proof); err != nil {
		return "", err
	}
	return proof, nil
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

// GetL2UnprocessedMessages fetch list of unprocessed messages
func (m *layer2MessageOrm) GetL2UnprocessedMessages() ([]*L2Message, error) {
	rows, err := m.db.Queryx(`SELECT nonce, height, sender, target, value, fee, gas_limit, deadline, calldata, layer2_hash FROM l2_message WHERE status = $1 ORDER BY nonce ASC;`, MsgPending)
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

	_, err := m.db.NamedExec(`INSERT INTO public.l2_message (nonce, height, sender, target, value, fee, gas_limit, deadline, calldata, layer2_hash) VALUES (:nonce, :height, :sender, :target, :value, :fee, :gas_limit, :deadline, :calldata, :layer2_hash);`, messageMaps)
	if err != nil {
		nonces := make([]uint64, 0, len(messages))
		heights := make([]*big.Int, 0, len(messages))
		for _, msg := range messages {
			nonces = append(nonces, msg.Nonce)
			heights = append(heights, new(big.Int).Set(msg.Height.ToInt()))
		}
		log.Error("failed to insert layer2Messages", "nonces", nonces, "heights", heights, "err", err)
	}
	return err
}

// UpdateLayer1Hash update corresponding layer1 hash given message nonce
func (m *layer2MessageOrm) UpdateLayer1Hash(ctx context.Context, layer2Hash string, layer1Hash string) error {
	if _, err := m.db.ExecContext(ctx, m.db.Rebind("update l2_message set layer1_hash = ? where layer2_hash = ?;"), layer1Hash, layer2Hash); err != nil {
		return err
	}

	return nil
}

// UpdateMessageProof update corresponding message proof given message nonce
func (m *layer2MessageOrm) UpdateMessageProof(ctx context.Context, layer2Hash, proof string) error {
	if _, err := m.db.ExecContext(ctx, m.db.Rebind("update l2_message set proof = ? where layer2_hash = ?;"), proof, layer2Hash); err != nil {
		return err
	}

	return nil
}

// UpdateLayer2Status updates message stauts
func (m *layer2MessageOrm) UpdateLayer2Status(ctx context.Context, layer2Hash string, status MsgStatus) error {
	if _, err := m.db.ExecContext(ctx, m.db.Rebind("update l2_message set status = ? where layer2_hash = ?;"), status, layer2Hash); err != nil {
		return err
	}

	return nil
}

// UpdateLayer2StatusAndLayer1Hash updates message stauts and layer1 transaction hash
func (m *layer2MessageOrm) UpdateLayer2StatusAndLayer1Hash(ctx context.Context, layer2Hash string, layer1Hash string, status MsgStatus) error {
	if _, err := m.db.ExecContext(ctx, m.db.Rebind("update l2_message set status = ?, layer1_hash = ? where layer2_hash = ?;"), status, layer1Hash, layer2Hash); err != nil {
		return err
	}

	return nil
}

// GetLayer2LatestWatchedHeight returns latest height stored in the table
func (m *layer2MessageOrm) GetLayer2LatestWatchedHeight() (*big.Int, error) {
	// @note It's not correct, since we may don't have message in some blocks.
	// But it will only be called at start, some redundancy is acceptable.
	row := m.db.QueryRow("SELECT COALESCE(MAX(height), -1) FROM l2_message;")

	var height bigint.Int = bigint.New(0)
	if err := row.Scan(&height); err != nil {
		return height.SetInt64(-1), err
	}
	if height.Cmp(big.NewInt(0)) < 0 {
		return height.SetInt64(-1), fmt.Errorf("could not get height due to database return negative")
	}
	return height.ToInt(), nil
}
