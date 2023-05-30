package orm

import (
	"encoding/json"

	"gorm.io/gorm"

	"scroll-tech/coordinator/internal/types"
)

// SessionInfo is structure of agg task message
type SessionInfo struct {
	db *gorm.DB `gorm:"-"`

	RollersInfo string `json:"rollers_info" gorm:"rollers_info"`
	Hash        string `json:"hash" gorm:"hash"`
}

// NewSessionInfo create an sessionInfoOrm instance
func NewSessionInfo(db *gorm.DB) *SessionInfo {
	return &SessionInfo{db: db}
}

// TableName define the SessionInfo table name
func (*SessionInfo) TableName() string {
	return "session_info"
}

// GetSessionInfosByHashes get the rollers info of session info by hash list
func (o *SessionInfo) GetSessionInfosByHashes(hashes []string) ([]types.RollersInfo, error) {
	if len(hashes) == 0 {
		return nil, nil
	}

	var sessionInfos []SessionInfo
	err := o.db.Select("rollers_info").Where("hash IN (?)", hashes).Find(&sessionInfos).Error
	if err != nil {
		return nil, err
	}

	var rollersInfoList []types.RollersInfo
	for _, sessionInfo := range sessionInfos {
		var rollersInfo types.RollersInfo
		if err := json.Unmarshal([]byte(sessionInfo.RollersInfo), &rollersInfo); err != nil {
			return nil, err
		}
		rollersInfoList = append(rollersInfoList, rollersInfo)
	}
	return rollersInfoList, nil
}

// InsertSessionInfo insert a session info record
func (o *SessionInfo) InsertSessionInfo(rollersInfo *types.RollersInfo) error {
	infoBytes, err := json.Marshal(rollersInfo)
	if err != nil {
		return err
	}
	err = o.db.Exec("INSERT INTO session_info (hash, rollers_info) VALUES (?, ?) ON CONFLICT (hash) DO UPDATE SET rollers_info = EXCLUDED.rollers_info", rollersInfo.ID, infoBytes).Error
	if err != nil {
		return err
	}
	return nil
}
