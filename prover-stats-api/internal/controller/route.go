package controller

import (
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
	"scroll-tech/prover-stats-api/internal/config"
	"scroll-tech/prover-stats-api/internal/logic"
	"scroll-tech/prover-stats-api/internal/middleware"
)

func Route(db *gorm.DB, port string, cfg *config.Config) {
	taskService := logic.NewProverTaskLogic(db)

	r := gin.Default()
	middleware.Secret = cfg.ApiSecret
	r.Use(middleware.JWTAuthMiddleware())
	r.GET("swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	router := r.Group("/api/v1")

	c := NewProverTaskController(router, taskService)
	c.Route()

	go func() {
		r.Run(port)
	}()
}
