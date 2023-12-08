package route

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"

	"scroll-tech/common/observability"

	"scroll-tech/bridge-history-api/internal/config"
	"scroll-tech/bridge-history-api/internal/controller/api"
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

	r.GET("/txs", api.HistoryCtrler.GetTxsByAddress)
	r.GET("/withdrawals", api.HistoryCtrler.GetL2WithdrawalsByAddress)
	r.GET("/claimablewithdrawals", api.HistoryCtrler.GetL2ClaimableWithdrawalsByAddress)

	r.POST("/txsbyhashes", api.HistoryCtrler.PostQueryTxsByHashes)
}
