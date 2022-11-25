package orm

import (
	"database/sql"
	"encoding/json"

	"github.com/jmoiron/sqlx"
)

type sessionInfoOrm struct {
	db *sqlx.DB
}

var _ SessionInfoOrm = (*sessionInfoOrm)(nil)

// NewSessionInfoOrm create an sessionInfoOrm instance
func NewSessionInfoOrm(db *sqlx.DB) SessionInfoOrm {
	return &sessionInfoOrm{db: db}
}

func (o *sessionInfoOrm) GetSessionInfoByID(id string) (*SessionInfo, error) {
	row := o.db.QueryRow(`SELECT rollers_info FROM session_info WHERE id = $1 and rollers_info IS NOT NULL`, id)
	var infoBytes []byte
	if err := row.Scan(&infoBytes); err != nil {
		if err == sql.ErrNoRows {
			return &SessionInfo{}, nil
		}
		return nil, err
	}
	rollersInfo := &SessionInfo{}
	if err := json.Unmarshal(infoBytes, rollersInfo); err != nil {
		return nil, err
	}
	return rollersInfo, nil
}

func (o *sessionInfoOrm) GetSessionInfosByIDs(ids []string) ([]*SessionInfo, error) {
	var sessionInfos []*SessionInfo
	for _, id := range ids {
		sessionInfo, err := o.GetSessionInfoByID(id)
		if err != nil {
			return nil, err
		}
		sessionInfos = append(sessionInfos, sessionInfo)
	}
	return sessionInfos, nil
}

func (o *sessionInfoOrm) SetSessionInfo(rollersInfo *SessionInfo) error {
	infoBytes, err := json.Marshal(rollersInfo)
	if err != nil {
		return err
	}
	sqlStr := "INSERT INTO session_info (id, rollers_info) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET rollers_info = EXCLUDED.rollers_info;"
	_, err = o.db.Exec(sqlStr, rollersInfo.ID, infoBytes)
	return err
}
