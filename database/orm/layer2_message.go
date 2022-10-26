package orm

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/scroll-tech/go-ethereum/log"
)

type layer2MessageOrm struct {
	db *sqlx.DB
}

var _ Layer2MessageOrm = (*layer2MessageOrm)(nil)

// NewLayer2MessageOrm create an Layer2MessageOrm instance
func NewLayer2MessageOrm(db *sqlx.DB) Layer2MessageOrm {
	return &layer2MessageOrm{db: db}
}

// GetLayer2MessageByNonce fetch message by nonce
func (m *layer2MessageOrm) GetLayer2MessageByNonce(nonce uint64) (*Layer2Message, error) {
	msg := Layer2Message{}

	var tempcontent []byte
	row := m.db.QueryRow(`SELECT content, height, layer2_hash, status FROM layer2_message WHERE nonce = $1`, nonce)
	if err := row.Scan(&tempcontent, &msg.Height, &msg.Layer2Hash, &msg.Status); err != nil {
		return nil, err
	}

	err := json.Unmarshal(tempcontent, &msg.Content)
	if err != nil {
		log.Error("failed to unmarshal layer2Messages content", "err", err)
		return nil, err
	}

	return &msg, nil
}

// GetLayer2MessageByLayer2Hash fetch message by layer2Hash
func (m *layer2MessageOrm) GetLayer2MessageByLayer2Hash(layer2Hash string) (*Layer2Message, error) {
	msg := Layer2Message{}

	var tempcontent []byte
	row := m.db.QueryRow(`SELECT content, height, layer2_hash, status FROM layer2_message WHERE layer2_hash = $1`, layer2Hash)
	if err := row.Scan(&tempcontent, &msg.Height, &msg.Layer2Hash, &msg.Status); err != nil {
		return nil, err
	}

	err := json.Unmarshal(tempcontent, &msg.Content)
	if err != nil {
		log.Error("failed to unmarshal layer2Messages content", "err", err)
		return nil, err
	}

	return &msg, nil
}

// GetMessageProofByLayer2Hash fetch message proof by layer2Hash
func (m *layer2MessageOrm) GetMessageProofByLayer2Hash(layer2Hash string) (string, error) {
	row := m.db.QueryRow(`SELECT proof FROM layer2_message WHERE layer2_hash = $1`, layer2Hash)
	var proof string
	if err := row.Scan(&proof); err != nil {
		return "", err
	}
	return proof, nil
}

// MessageProofExistByLayer2Hash fetch message by layer2Hash
func (m *layer2MessageOrm) MessageProofExistByLayer2Hash(layer2Hash string) (bool, error) {
	err := m.db.QueryRow(`SELECT layer2_hash FROM layer2_message WHERE layer2_hash = $1 and proof IS NOT NULL`, layer2Hash).Scan(&layer2Hash)
	if err != nil {
		if err != sql.ErrNoRows {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

// GetL2UnprocessedMessages fetch list of unprocessed messages
func (m *layer2MessageOrm) GetL2UnprocessedMessages() ([]*Layer2Message, error) {
	rows, err := m.db.Queryx(`SELECT content, height, layer2_hash FROM layer2_message WHERE status = $1 ORDER BY nonce ASC;`, MsgPending)
	if err != nil {
		return nil, err
	}

	var msgs []*Layer2Message
	for rows.Next() {
		msg := &Layer2Message{}
		var tempcontent []byte
		if err := rows.Scan(&tempcontent, &msg.Height, &msg.Layer2Hash, &msg.Status); err != nil {
			return nil, err
		}

		err := json.Unmarshal(tempcontent, &msg.Content)
		if err != nil {
			log.Error("failed to unmarshal layer2Messages content", "err", err)
			return nil, err
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

// SaveLayer2Messages batch save a list of layer2 messages
func (m *layer2MessageOrm) SaveLayer2Messages(ctx context.Context, messages []*Layer2Message) error {
	messageMaps := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		content, err := json.Marshal(msg.Content)
		if err != nil {
			log.Error("failed to marshal layer2Messages content", "err", err)
			return err
		}
		messageMaps[i] = map[string]interface{}{
			"content":     content,
			"height":      msg.Height,
			"layer2_hash": msg.Layer2Hash,
		}
	}

	_, err := m.db.NamedExec(`INSERT INTO public.layer2_message (content, height, layer2_hash) VALUES (:content, :height, :layer2_hash);`, messageMaps)
	if err != nil {
		log.Error("failed to insert layer2Messages", "err", err)
	}
	return err
}

// UpdateLayer1Hash update corresponding layer1 hash given message nonce
func (m *layer2MessageOrm) UpdateLayer1Hash(ctx context.Context, layer2Hash string, layer1Hash string) error {
	if _, err := m.db.ExecContext(ctx, m.db.Rebind("update layer2_message set layer1_hash = ? where layer2_hash = ?;"), layer1Hash, layer2Hash); err != nil {
		return err
	}

	return nil
}

// UpdateMessageProof update corresponding message proof given message nonce
func (m *layer2MessageOrm) UpdateMessageProof(ctx context.Context, layer2Hash, proof string) error {
	if _, err := m.db.ExecContext(ctx, m.db.Rebind("update layer2_message set proof = ? where layer2_hash = ?;"), proof, layer2Hash); err != nil {
		return err
	}

	return nil
}

// UpdateLayer2Status updates message stauts
func (m *layer2MessageOrm) UpdateLayer2Status(ctx context.Context, layer2Hash string, status MsgStatus) error {
	if _, err := m.db.ExecContext(ctx, m.db.Rebind("update layer2_message set status = ? where layer2_hash = ?;"), status, layer2Hash); err != nil {
		return err
	}

	return nil
}

// UpdateLayer2StatusAndLayer1Hash updates message stauts and layer1 transaction hash
func (m *layer2MessageOrm) UpdateLayer2StatusAndLayer1Hash(ctx context.Context, layer2Hash string, layer1Hash string, status MsgStatus) error {
	if _, err := m.db.ExecContext(ctx, m.db.Rebind("update layer2_message set status = ?, layer1_hash = ? where layer2_hash = ?;"), status, layer1Hash, layer2Hash); err != nil {
		return err
	}

	return nil
}

// GetLayer2LatestWatchedHeight returns latest height stored in the table
func (m *layer2MessageOrm) GetLayer2LatestWatchedHeight() (int64, error) {
	// @note It's not correct, since we may don't have message in some blocks.
	// But it will only be called at start, some redundancy is acceptable.
	row := m.db.QueryRow("SELECT COALESCE(MAX(height), -1) FROM layer2_message;")

	var height int64
	if err := row.Scan(&height); err != nil {
		return -1, err
	}
	if height < 0 {
		return -1, fmt.Errorf("could not get height due to database return negative")
	}
	return height, nil
}
