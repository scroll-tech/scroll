package provertask

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/orm"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

// ProverTask the interface of a collector who send data to prover
type ProverTask interface {
	Assign(ctx *gin.Context, getTaskParameter *coordinatorType.GetTaskParameter) (*coordinatorType.GetTaskSchema, error)
}

// BaseProverTask a base prover task which contain series functions
type BaseProverTask struct {
	cfg *config.Config
	db  *gorm.DB

	batchOrm      *orm.Batch
	chunkOrm      *orm.Chunk
	blockOrm      *orm.L2Block
	proverTaskOrm *orm.ProverTask
}
