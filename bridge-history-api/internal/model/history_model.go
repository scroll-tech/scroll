package model

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"bridge-history-api/internal/logic"
)

// QueryByAddressRequest the request parameter of address api
type QueryByAddressRequest struct {
	Address string `form:"address"`
	Offset  int    `form:"offset"`
	Limit   int    `form:"limit"`
}

// QueryByHashRequest the request parameter of hash api
type QueryByHashRequest struct {
	Txs []string `form:"txs"`
}

// ResultData contains return txs and total
type ResultData struct {
	Result []*logic.TxHistoryInfo `json:"result"`
	Total  uint64                 `json:"total"`
}

// Response the response schema
type Response struct {
	ErrCode int         `json:"errcode"`
	ErrMsg  string      `json:"errmsg"`
	Data    interface{} `json:"data"`
}

// RenderJSON renders response with json
func RenderJSON(ctx *gin.Context, errCode int, err error, data interface{}) {
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}
	renderData := Response{
		ErrCode: errCode,
		ErrMsg:  errMsg,
		Data:    data,
	}
	ctx.JSON(http.StatusOK, renderData)
}
