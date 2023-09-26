package observability

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"bridge-history-api/internal/types"
	"bridge-history-api/utils"
)

// ProbesController probe check controller
type ProbesController struct {
	db *gorm.DB
}

// NewProbesController returns an ProbesController instance
func NewProbesController(db *gorm.DB) *ProbesController {
	return &ProbesController{
		db: db,
	}
}

// HealthCheck the api controller for health check
func (a *ProbesController) HealthCheck(c *gin.Context) {
	if _, err := utils.Ping(a.db); err != nil {
		types.RenderFatal(c, err)
		return
	}
	types.RenderSuccess(c, nil)
}

// Ready the api controller for ready check
func (a *ProbesController) Ready(c *gin.Context) {
	types.RenderSuccess(c, nil)
}
