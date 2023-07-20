package orm

import (
	"time"

	"gorm.io/gorm"
)

// AssetType can be ETH/ERC20/ERC1155/ERC721
type AssetType int

// MsgType can be layer1/layer2 msg
type MsgType int

func (a AssetType) String() string {
	switch a {
	case ETH:
		return "ETH"
	case ERC20:
		return "ERC20"
	case ERC1155:
		return "ERC1155"
	case ERC721:
		return "ERC721"
	}
	return "Unknown Asset Type"
}

const (
	// ETH = 0
	ETH AssetType = iota
	// ERC20 = 1
	ERC20
	// ERC721 = 2
	ERC721
	// ERC1155 = 3
	ERC1155
)

const (
	// UnknownMsg = 0
	UnknownMsg MsgType = iota
	// Layer1Msg = 1
	Layer1Msg
	// Layer2Msg = 2
	Layer2Msg
)

// CrossMsg represents a cross message from layer 1 to layer 2
type CrossMsg struct {
	db *gorm.DB `gorm:"column:-"`

	ID           uint64         `json:"id" gorm:"column:id"`
	MsgHash      string         `json:"msg_hash" gorm:"column:msg_hash"`
	Height       uint64         `json:"height" gorm:"column:height"`
	Sender       string         `json:"sender" gorm:"column:sender"`
	Target       string         `json:"target" gorm:"column:target"`
	Amount       string         `json:"amount" gorm:"column:amount"`
	Layer1Hash   string         `json:"layer1_hash" gorm:"column:layer1_hash;default:''"`
	Layer2Hash   string         `json:"layer2_hash" gorm:"column:layer2_hash;default:''"`
	Layer1Token  string         `json:"layer1_token" gorm:"column:layer1_token;default:''"`
	Layer2Token  string         `json:"layer2_token" gorm:"column:layer2_token;default:''"`
	TokenIDs     string         `json:"token_ids" gorm:"column:token_ids;default:''"`
	TokenAmounts string         `json:"token_amounts" gorm:"column:token_amounts;default:''"`
	Asset        int            `json:"asset" gorm:"column:asset"`
	MsgType      int            `json:"msg_type" gorm:"column:msg_type"`
	Timestamp    *time.Time     `json:"timestamp" gorm:"column:block_timestamp;default;NULL"`
	CreatedAt    *time.Time     `json:"created_at" gorm:"column:created_at"`
	UpdatedAt    *time.Time     `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;default:NULL"`
}

// TableName returns the table name for the Batch model.
func (*CrossMsg) TableName() string {
	return "cross_message"
}
