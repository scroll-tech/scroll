package route

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"bridge-history-api/config"
	"bridge-history-api/internal/controller"
)

// Route routes the APIs
func Route(router *gin.Engine, conf *config.Config) {
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r := router.Group("api/")
	r.POST("/txsbyhashes", controller.HistoryCtrler.PostQueryTxsByHash)
	r.GET("/claimable", controller.HistoryCtrler.GetAllClaimableTxsByAddr)
}
