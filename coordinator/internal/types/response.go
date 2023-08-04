package types

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response the response schema
type Response struct {
	ErrCode int         `json:"errcode,omitempty"`
	ErrMsg  string      `json:"errmsg,omitempty"`
	Data    interface{} `json:"data,omitempty"`
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
