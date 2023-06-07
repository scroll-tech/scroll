package orm

import (
	"encoding/json"

	"github.com/jmoiron/sqlx"

	"scroll-tech/common/types"
)

type sessionInfoOrm struct {
	db *sqlx.DB
}

var _ SessionInfoOrm = (*sessionInfoOrm)(nil)

// NewSessionInfoOrm create an sessionInfoOrm instance
func NewSessionInfoOrm(db *sqlx.DB) SessionInfoOrm {
	return &sessionInfoOrm{db: db}
}

func (o *sessionInfoOrm) GetSessionInfosByHashes(hashes []string) ([]*types.SessionInfo, error) {
	if len(hashes) == 0 {
		return nil, nil
	}
	query, args, err := sqlx.In("SELECT rollers_info FROM session_info WHERE hash IN (?);", hashes)
	if err != nil {
		return nil, err
	}
	rows, errQuery := o.db.Queryx(o.db.Rebind(query), args...)
	if errQuery != nil {
		return nil, errQuery
	}
	defer func() { _ = rows.Close() }()

	var sessionInfos []*types.SessionInfo
	for rows.Next() {
		var infoBytes []byte
		if err := rows.Scan(&infoBytes); err != nil {
			return nil, err
		}
		sessionInfo := &types.SessionInfo{}
		if err := json.Unmarshal(infoBytes, sessionInfo); err != nil {
			return nil, err
		}
		sessionInfos = append(sessionInfos, sessionInfo)
	}
	if errQuery = rows.Err(); errQuery != nil {
		return nil, errQuery
	}

	return sessionInfos, nil
}

func (o *sessionInfoOrm) SetSessionInfo(rollersInfo *types.SessionInfo) error {
	infoBytes, err := json.Marshal(rollersInfo)
	if err != nil {
		return err
	}
	sqlStr := "INSERT INTO session_info (hash, rollers_info) VALUES ($1, $2) ON CONFLICT (hash) DO UPDATE SET rollers_info = EXCLUDED.rollers_info;"
	_, err = o.db.Exec(sqlStr, rollersInfo.ID, infoBytes)
	return err
}
