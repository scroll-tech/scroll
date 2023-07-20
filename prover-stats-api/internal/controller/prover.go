package controller

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"scroll-tech/prover-stats-api/internal/logic"
)

type ProverController struct {
	logic *logic.ProverLogic
}

func NewProverController(db *gorm.DB) *ProverController {
	return &ProverController{logic: logic.NewProverLogic(db)}
}

func (p *ProverTaskController) ListProvers(ctx *gin.Context) {

}
