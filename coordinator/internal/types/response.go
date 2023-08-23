package types

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"scroll-tech/common/types"
)

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
		if errCode == types.ErrCoordinatorGetTaskFailure || errCode == types.ErrCoordinatorHandleZkProofFailure {
			errMsg = "Internal Server Error"
		} else {
			errMsg = err.Error()
		}
	}
	renderData := Response{
		ErrCode: errCode,
		ErrMsg:  errMsg,
		Data:    data,
	}
	ctx.Set("errcode", errCode)
	ctx.JSON(http.StatusOK, renderData)
}
