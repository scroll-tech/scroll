package orm

import (
	"context"

	"github.com/jmoiron/sqlx"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
)

type submissionInfoOrm struct {
	db *sqlx.DB
}

var _ SubmissionInfoOrm = (*submissionInfoOrm)(nil)

// NewSubmissionInfoOrm create an submissionInfoOrm instance
func NewSubmissionInfoOrm(db *sqlx.DB) SubmissionInfoOrm {
	return &submissionInfoOrm{db: db}
}

func (o *submissionInfoOrm) GetSubmissionInfosByHashes(hashes []string) ([]*types.SubmissionInfo, error) {
	if len(hashes) == 0 {
		return nil, nil
	}
	query, args, err := sqlx.In("SELECT * FROM submission_info WHERE task_id IN (?);", hashes)
	if err != nil {
		return nil, err
	}
	rows, err := o.db.Queryx(o.db.Rebind(query), args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var submissionInfos []*types.SubmissionInfo
	for rows.Next() {
		var submissionInfo types.SubmissionInfo
		if err = rows.StructScan(&submissionInfo); err != nil {
			return nil, err
		}
		submissionInfos = append(submissionInfos, &submissionInfo)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return submissionInfos, nil
}

func (o *submissionInfoOrm) GetSubmissionInfosByRoller(pubKey string) ([]*types.SubmissionInfo, error) {
	rows, err := o.db.Queryx("SELECT * FROM submission_info WHERE roller_public_key = $1;", pubKey)
	if err != nil {
		return nil, err
	}

	var subs []*types.SubmissionInfo
	for rows.Next() {
		var sub types.SubmissionInfo
		err = rows.StructScan(&sub)
		if err != nil {
			return nil, err
		}
		subs = append(subs, &sub)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return subs, nil
}

func (o *submissionInfoOrm) SetSubmissionInfo(rollersInfo *types.SubmissionInfo) error {
	sqlStr := "INSERT INTO submission_info (task_id, roller_public_key, prove_type, roller_name, proving_status, failure_type, reward, proof, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) ON CONFLICT (task_id, roller_public_key) DO UPDATE SET proving_status = EXCLUDED.proving_status;"
	_, err := o.db.Exec(sqlStr, rollersInfo.TaskID, rollersInfo.RollerPublicKey, rollersInfo.ProveType, rollersInfo.RollerName,
		rollersInfo.ProvingStatus, rollersInfo.FailureType, rollersInfo.Reward, rollersInfo.Proof, rollersInfo.CreatedAt)
	return err
}

// UpdateSubmissionInfoProvingStatus update the submission info proving status
func (o *submissionInfoOrm) UpdateSubmissionInfoProvingStatus(ctx context.Context, proveType message.ProveType, taskID string, pk string, status types.RollerProveStatus) error {
	if _, err := o.db.ExecContext(ctx, o.db.Rebind("update submission_info set proving_status = ? where prove_type = ? and task_id = ? and roller_public_key = ? ;"), int(proveType), int(status), taskID, pk); err != nil {
		return err
	}
	return nil
}
