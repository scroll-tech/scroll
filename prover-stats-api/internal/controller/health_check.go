package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"scroll-tech/common/database"
	"scroll-tech/common/types"
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
	if _, err := database.Ping(a.db); err != nil {
		types.RenderFatal(c, http.StatusInternalServerError, types.InternalServerError, nil, nil)
		return
	}
	types.RenderJSON(c, types.Success, nil, nil)
}
