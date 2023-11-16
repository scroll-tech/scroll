package route

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"

	"bridge-history-api/config"
	"bridge-history-api/internal/controller"
	"bridge-history-api/observability"
)

// Route routes the APIs
func Route(router *gin.Engine, conf *config.Config, reg prometheus.Registerer) {
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	observability.Use(router, "bridge_history_api", reg)

	r := router.Group("api/")
	r.GET("/txs", controller.HistoryCtrler.GetTxsByAddress)
	r.GET("/withdrawals", controller.HistoryCtrler.GetL2WithdrawalsByAddress)
	r.GET("/claimablewithdrawals", controller.HistoryCtrler.GetL2ClaimableWithdrawalsByAddress)

	r.POST("/txsbyhashes", controller.HistoryCtrler.PostQueryTxsByHashes)
}
