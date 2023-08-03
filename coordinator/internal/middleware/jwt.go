package middleware

import (
	"errors"
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"

	"scroll-tech/common/types"

	coordinatorType "scroll-tech/coordinator/internal/types"
)

func unauthorized(c *gin.Context, _ int, message string) {
	var errCode int
	err := errors.New(message)
	switch message {
	case jwt.ErrExpiredToken.Error():
		errCode = types.ErrJWTTokenExpired
	default:
		errCode = types.ErrJWTCommonErr
	}
	coordinatorType.RenderJSON(c, errCode, err, nil)
}

func loginResponse(c *gin.Context, code int, message string, time time.Time) {
	resp := coordinatorType.LoginSchema{
		Time:  time,
		Token: message,
	}
	coordinatorType.RenderJSON(c, types.Success, nil, resp)
}
