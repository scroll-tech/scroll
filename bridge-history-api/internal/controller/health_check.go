package controller

import (
	"net/http"

	"bridge-history-api/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"bridge-history-api/internal/types"
)

// HealthCheckController is health check API
type HealthCheckController struct {
	db *gorm.DB
}

// NewHealthCheckController returns an HealthCheckController instance
func NewHealthCheckController(db *gorm.DB) *HealthCheckController {
	return &HealthCheckController{
		db: db,
	}
}

// HealthCheck the api controller for coordinator health check
func (a *HealthCheckController) HealthCheck(c *gin.Context) {
	if _, err := utils.Ping(a.db); err != nil {
		types.RenderFatal(c, http.StatusInternalServerError, types.InternalServerError, nil, nil)
		return
	}
	types.RenderJSON(c, types.Success, nil, nil)
}
