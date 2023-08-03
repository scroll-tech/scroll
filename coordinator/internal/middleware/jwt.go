package middleware

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"scroll-tech/coordinator/internal/types"
)

func unauthorized(c *gin.Context, code int, message string) {
	err := errors.New(message)
	types.RenderJSON(c, types.ErrJWTAuthFailure, err, nil)
}

func loginResponse(c *gin.Context, code int, message string, time time.Time) {
	resp := types.LoginSchema{
		Time:  time,
		Token: message,
	}
	types.RenderJSON(c, code, nil, resp)
}
