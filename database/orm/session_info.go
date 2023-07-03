package orm

import (
	"context"

	"github.com/jmoiron/sqlx"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
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
	query, args, err := sqlx.In("SELECT * FROM session_info WHERE task_id IN (?);", hashes)
	if err != nil {
		return nil, err
	}
	rows, err := o.db.Queryx(o.db.Rebind(query), args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var sessionInfos []*types.SessionInfo
	for rows.Next() {
		var sessionInfo types.SessionInfo
		if err = rows.StructScan(&sessionInfo); err != nil {
			return nil, err
		}
		sessionInfos = append(sessionInfos, &sessionInfo)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return sessionInfos, nil
}

func (o *sessionInfoOrm) SetSessionInfo(rollersInfo *types.SessionInfo) error {
	sqlStr := "INSERT INTO session_info (task_id, roller_public_key, prove_type, roller_name, proving_status, failure_type, reward, proof, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) ON CONFLICT (task_id, roller_public_key) DO UPDATE SET proving_status = EXCLUDED.proving_status;"
	_, err := o.db.Exec(sqlStr, rollersInfo.TaskID, rollersInfo.RollerPublicKey, rollersInfo.ProveType, rollersInfo.RollerName,
		rollersInfo.ProvingStatus, rollersInfo.FailureType, rollersInfo.Reward, rollersInfo.Proof, rollersInfo.CreatedAt)
	return err
}

// UpdateSessionInfoProvingStatus update the session info proving status
func (o *sessionInfoOrm) UpdateSessionInfoProvingStatus(ctx context.Context, proveType message.ProveType, taskID string, pk string, status types.RollerProveStatus) error {
	if _, err := o.db.ExecContext(ctx, o.db.Rebind("update session_info set proving_status = ? where prove_type = ? and task_id = ? and roller_public_key = ? ;"), int(proveType), int(status), taskID, pk); err != nil {
		return err
	}
	return nil
}
