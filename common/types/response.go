package types

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"
)

// Response the response schema
type Response struct {
	ErrCode int         `json:"errcode"`
	ErrMsg  string      `json:"errmsg"`
	Data    interface{} `json:"data"`
}

func (resp *Response) DecodeData(out interface{}) error {
	// Decode generically unmarshaled JSON (map[string]any, []any) into a typed struct
	// honoring `json` tags and allowing weak type conversions.
	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "json",
		Result:  out,
	})
	if err != nil {
		return err
	}
	return dec.Decode(resp.Data)
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

// RenderSuccess renders success response with json
func RenderSuccess(ctx *gin.Context, data interface{}) {
	RenderJSON(ctx, Success, nil, data)
}

// RenderFailure renders failure response with json
func RenderFailure(ctx *gin.Context, errCode int, err error) {
	RenderJSON(ctx, errCode, err, nil)
}

// RenderFatal renders fatal response with json
func RenderFatal(ctx *gin.Context, err error) {
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}
	renderData := Response{
		ErrCode: InternalServerError,
		ErrMsg:  errMsg,
		Data:    nil,
	}
	ctx.Set("errcode", InternalServerError)
	ctx.JSON(http.StatusInternalServerError, renderData)
}
