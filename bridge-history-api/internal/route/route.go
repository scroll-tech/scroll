package route

import (
	"github.com/gin-gonic/gin"

	"bridge-history-api/config"
	"bridge-history-api/internal/controller"
)

// Route routes the APIs
func Route(router *gin.Engine, conf *config.Config) {

	r := router.Group("api/")

	v1(r, conf)
}

func v1(r *gin.RouterGroup, conf *config.Config) {
	r.GET("/txs", controller.HistoryCtrler.GetAllTxsByAddr)
	r.GET("/txsbyhashes", controller.HistoryCtrler.PostQueryTxsByHash)
	r.GET("/claimable", controller.HistoryCtrler.GetAllClaimableTxsByAddr)

}
