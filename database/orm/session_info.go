package orm

import (
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/jmoiron/sqlx"
)

type sessionInfoOrm struct {
	db *sqlx.DB
}

var _ SessionInfoOrm = (*sessionInfoOrm)(nil)

// NewBlockBatchOrm create an blockBatchOrm instance
func NewSessionInfoOrm(db *sqlx.DB) SessionInfoOrm {
	return &sessionInfoOrm{db: db}
}

func (o *sessionInfoOrm) GetRollersInfoByID(id string) (*RollersInfo, error) {
	row := o.db.QueryRow(`SELECT rollers_info FROM session_info WHERE id = $1 and rollers_info IS NOT NULL`, id)
	var infoBytes []byte
	if err := row.Scan(&infoBytes); err != nil {
		if err == sql.ErrNoRows {
			return &RollersInfo{}, nil
		}
		return nil, err
	}
	rollersInfo := &RollersInfo{}
	if err := json.Unmarshal(infoBytes, rollersInfo); err != nil {
		return nil, err
	}
	return rollersInfo, nil
}

func (o *sessionInfoOrm) GetProvingSessionIDs() ([]string, error) {
	rows, err := o.db.Queryx(`SELECT id FROM block_batch WHERE proving_status = $1 OR proving_status = $2`, ProvingTaskAssigned, ProvingTaskProved)
	if err != nil {
		return nil, err
	}

	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			break
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 || errors.Is(err, sql.ErrNoRows) {
		// log.Warn("no pending batches in db", "err", err)
	} else if err != nil {
		return nil, err
	}

	return ids, rows.Close()
}

func (o *sessionInfoOrm) GetAllRollersInfo() ([]*RollersInfo, error) {
	ids, err := o.GetProvingSessionIDs()
	if err != nil {
		return nil, err
	}
	var rollersInfos []*RollersInfo
	for _, id := range ids {
		rollersInfo, err := o.GetRollersInfoByID(id)
		if err != nil {
			return nil, err
		}
		rollersInfos = append(rollersInfos, rollersInfo)
	}
	return rollersInfos, nil
}

func (o *sessionInfoOrm) SetRollersInfoByID(id string, rollersInfo *RollersInfo) error {
	infoBytes, err := json.Marshal(rollersInfo)
	if err != nil {
		return err
	}
	sqlStr := "INSERT INTO session_info (id, rollers_info) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET rollers_info = EXCLUDED.rollers_info;"
	_, err = o.db.Exec(sqlStr, id, infoBytes)
	return err
}

func (o *sessionInfoOrm) UpdateRollerProofStatusByID(id string, rollerPublicKey string, rollerProofStatus RollerProveStatus) error {
	row := o.db.QueryRow(`SELECT rollers_info FROM session_info WHERE id = $1 and rollers_info IS NOT NULL`, id)
	var infoBytes []byte
	if err := row.Scan(&infoBytes); err != nil {
		return err
	}
	var rollersInfo RollersInfo
	if err := json.Unmarshal(infoBytes, &rollersInfo); err != nil {
		return err
	}
	rollersInfo.RollerStatus[rollerPublicKey] = rollerProofStatus
	infoBytes, err := json.Marshal(rollersInfo)
	if err != nil {
		return err
	}
	_, err = o.db.Exec(o.db.Rebind(`UPDATE session_info set rollers_info = ? where id = ?;`), infoBytes, id)
	return err
}

func (o *sessionInfoOrm) DeleteRollersInfoByID(id string) error {
	_, err := o.db.Exec(o.db.Rebind(`UPDATE session_info set rollers_info = ? where id = ?;`), sql.NullString{}, id)
	return err
}
