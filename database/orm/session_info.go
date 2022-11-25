package orm

import (
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

func (o *sessionInfoOrm) GetSessionInfosByIDs(ids []string) ([]*SessionInfo, error) {
	if len(ids) == 0 {
		return []*SessionInfo{}, nil
	}
	query, args, err := sqlx.In("SELECT rollers_info FROM session_info WHERE id IN (?);", ids)
	if err != nil {
		return nil, err
	}
	rows, err := o.db.Queryx(o.db.Rebind(query), args...)
	if err != nil {
		return nil, err
	}
	var sessionInfos []*SessionInfo
	for rows.Next() {
		var infoBytes []byte
		if err := rows.Scan(&infoBytes); err != nil {
			return nil, err
		}
		sessionInfo := &SessionInfo{}
		if err := json.Unmarshal(infoBytes, sessionInfo); err != nil {
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
