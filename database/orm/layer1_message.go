package orm

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/scroll-tech/go-ethereum/log"
)

type layer1MessageOrm struct {
	db *sqlx.DB
}

var _ Layer1MessageOrm = (*layer1MessageOrm)(nil)

// NewLayer1MessageOrm create an Layer1MessageOrm instance
func NewLayer1MessageOrm(db *sqlx.DB) Layer1MessageOrm {
	return &layer1MessageOrm{db: db}
}

// GetLayer1MessageByLayer1Hash fetch message by nonce
func (m *layer1MessageOrm) GetLayer1MessageByLayer1Hash(layer1Hash string) (*Layer1Message, error) {
	msg := Layer1Message{}

	row := m.db.QueryRow(`SELECT content, height, layer1_hash, status FROM layer1_message WHERE layer1_hash = $1`, layer1Hash)

	var tempcontent []byte
	if err := row.Scan(&tempcontent, &msg.Height, &msg.Layer1Hash, &msg.Status); err != nil {
		return nil, err
	}

	err := json.Unmarshal(tempcontent, &msg.Content)
	if err != nil {
		log.Error("failed to unmarshal layer1Messages content", "err", err)
		return nil, err
	}

	return &msg, nil
}

// GetLayer1MessageByNonce fetch message by nonce
func (m *layer1MessageOrm) GetLayer1MessageByNonce(nonce uint64) (*Layer1Message, error) {
	msg := Layer1Message{}

	row := m.db.QueryRow(`SELECT content, height, layer1_hash, status FROM layer1_message WHERE nonce = $1`, nonce)

	var tempcontent []byte
	if err := row.Scan(&tempcontent, &msg.Height, &msg.Layer1Hash, &msg.Status); err != nil {
		return nil, err
	}
	err := json.Unmarshal(tempcontent, &msg.Content)
	if err != nil {
		log.Error("failed to unmarshal layer1Messages content", "err", err)
		return nil, err
	}
	return &msg, nil
}

// GetL1UnprocessedMessages fetch list of unprocessed messages
func (m *layer1MessageOrm) GetL1UnprocessedMessages() ([]*Layer1Message, error) {
	rows, err := m.db.Queryx(`SELECT content, height, layer1_hash, status FROM layer1_message WHERE status = $1 ORDER BY nonce ASC;`, MsgPending)
	if err != nil {
		return nil, err
	}

	var msgs []*Layer1Message
	for rows.Next() {
		msg := &Layer1Message{}
		var tempcontent []byte
		if err := rows.Scan(&tempcontent, &msg.Height, &msg.Layer1Hash, &msg.Status); err != nil {
			return nil, err
		}
		err := json.Unmarshal(tempcontent, &msg.Content)
		if err != nil {
			log.Error("failed to unmarshal layer1Messages content", "err", err)
			return nil, err
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

// SaveLayer1Messages batch save a list of layer1 messages
func (m *layer1MessageOrm) SaveLayer1Messages(ctx context.Context, messages []*Layer1Message) error {
	messageMaps := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		content, err := json.Marshal(msg.Content)
		if err != nil {
			log.Error("failed to marshal msg content", "err", err)
			return err
		}
		messageMaps[i] = map[string]interface{}{
			"content":     content,
			"height":      msg.Height,
			"layer1_hash": msg.Layer1Hash,
		}
	}
	_, err := m.db.NamedExec(`INSERT INTO public.layer1_message (content, height, layer1_hash) VALUES (:content, :height, :layer1_hash);`, messageMaps)
	if err != nil {
		log.Error("failed to insert layer1Messages", "err", err)
	}
	return err
}

// UpdateLayer2Hash update corresponding layer2 hash given message nonce
func (m *layer1MessageOrm) UpdateLayer2Hash(ctx context.Context, layer1Hash string, layer2Hash string) error {
	if _, err := m.db.ExecContext(ctx, m.db.Rebind("update layer1_message set layer2_hash = ? where layer1_hash = ?;"), layer2Hash, layer1Hash); err != nil {
		return err
	}

	return nil
}

// UpdateLayer1Status updates message stauts
func (m *layer1MessageOrm) UpdateLayer1Status(ctx context.Context, layer1Hash string, status MsgStatus) error {
	if _, err := m.db.ExecContext(ctx, m.db.Rebind("update layer1_message set status = ? where layer1_hash = ?;"), status, layer1Hash); err != nil {
		return err
	}

	return nil
}

// UpdateLayer1StatusAndLayer2Hash updates message status and layer2 transaction hash
func (m *layer1MessageOrm) UpdateLayer1StatusAndLayer2Hash(ctx context.Context, layer1Hash, layer2Hash string, status MsgStatus) error {
	if _, err := m.db.ExecContext(ctx, m.db.Rebind("update layer1_message set status = ?, layer2_hash = ? where layer1_hash = ?;"), status, layer2Hash, layer1Hash); err != nil {
		return err
	}

	return nil
}

// GetLayer1LatestWatchedHeight returns latest height stored in the table
func (m *layer1MessageOrm) GetLayer1LatestWatchedHeight() (int64, error) {
	// @note It's not correct, since we may don't have message in some blocks.
	// But it will only be called at start, some redundancy is acceptable.
	row := m.db.QueryRow("SELECT MAX(height) FROM layer1_message;")

	var height int64
	if err := row.Scan(&height); err != nil {
		if err == sql.ErrNoRows {
			return -1, nil
		}
		return 0, err
	}
	return height, nil
}
