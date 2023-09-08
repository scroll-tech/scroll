package observability

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	
	"scroll-tech/common/database"
	"scroll-tech/common/types"
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
	if _, err := database.Ping(a.db); err != nil {
		types.RenderFatal(c, http.StatusInternalServerError, types.InternalServerError, nil, nil)
		return
	}
	types.RenderJSON(c, types.Success, nil, nil)
}

// Ready the api controller for ready check
func (a *ProbesController) Ready(c *gin.Context) {
	types.RenderJSON(c, types.Success, nil, nil)
}
