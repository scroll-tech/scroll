package proxy

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"scroll-tech/coordinator/internal/types"
)

type proverDataPersist struct {
	db *gorm.DB
}

// NewProverDataPersist creates a persistence instance backed by a gorm DB.
func NewProverDataPersist(db *gorm.DB) *proverDataPersist {
	return &proverDataPersist{db: db}
}

// gorm model mapping to table `prover_sessions`
type proverSessionRecord struct {
	PublicKey string    `gorm:"column:public_key;not null"`
	Upstream  string    `gorm:"column:upstream;not null"`
	UpToken   string    `gorm:"column:up_token;not null"`
	Expired   time.Time `gorm:"column:expired;not null"`
}

func (proverSessionRecord) TableName() string { return "prover_sessions" }

// priority_upstream model
type priorityUpstreamRecord struct {
	PublicKey string `gorm:"column:public_key;not null"`
	Upstream  string `gorm:"column:upstream;not null"`
}

func (priorityUpstreamRecord) TableName() string { return "priority_upstream" }

// get retrieves ProverSession for a given user key, returns empty if still not exists
func (p *proverDataPersist) Get(userKey string) (*proverSession, error) {
	if p == nil || p.db == nil {
		return nil, nil
	}

	var rows []proverSessionRecord
	if err := p.db.Where("public_key = ?", userKey).Find(&rows).Error; err != nil || len(rows) == 0 {
		return nil, err
	}

	ret := &proverSession{
		proverToken: make(map[string]loginToken),
	}
	for _, r := range rows {
		ls := &types.LoginSchema{
			Token: r.UpToken,
			Time:  r.Expired,
		}
		ret.proverToken[r.Upstream] = loginToken{LoginSchema: ls}
	}
	return ret, nil
}

func (p *proverDataPersist) Update(userKey, up string, login *types.LoginSchema) error {
	if p == nil || p.db == nil || login == nil {
		return nil
	}

	rec := proverSessionRecord{
		PublicKey: userKey,
		Upstream:  up,
		UpToken:   login.Token,
		Expired:   login.Time,
	}

	return p.db.Clauses(
		clause.OnConflict{
			Columns:   []clause.Column{{Name: "public_key"}, {Name: "upstream"}},
			DoUpdates: clause.AssignmentColumns([]string{"up_token", "expired"}),
		},
	).Create(&rec).Error
}

type proverPriorityPersist struct {
	db *gorm.DB
}

func NewProverPriorityPersist(db *gorm.DB) *proverPriorityPersist {
	return &proverPriorityPersist{db: db}
}

func (p *proverPriorityPersist) Get(userKey string) (string, error) {
	if p == nil || p.db == nil {
		return "", nil
	}
	var rec priorityUpstreamRecord
	if err := p.db.Where("public_key = ?", userKey).First(&rec).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return "", err
		} else {
			return "", nil
		}

	}
	return rec.Upstream, nil
}

func (p *proverPriorityPersist) Update(userKey, up string) error {
	if p == nil || p.db == nil {
		return nil
	}
	rec := priorityUpstreamRecord{PublicKey: userKey, Upstream: up}
	return p.db.Clauses(
		clause.OnConflict{
			Columns:   []clause.Column{{Name: "public_key"}},
			DoUpdates: clause.Assignments(map[string]interface{}{"upstream": up}),
		},
	).Create(&rec).Error
}

func (p *proverPriorityPersist) Del(userKey string) error {
	if p == nil || p.db == nil {
		return nil
	}
	return p.db.Where("public_key = ?", userKey).Delete(&priorityUpstreamRecord{}).Error
}
