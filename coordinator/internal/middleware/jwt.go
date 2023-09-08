package middleware

import (
	"errors"
	"strings"
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"

	"scroll-tech/common/types"

	coordinatorType "scroll-tech/coordinator/internal/types"
)

func unauthorized(c *gin.Context, _ int, message string) {
	lower := strings.ToLower(message)
	var errCode int
	err := errors.New(lower)
	if jwt.ErrExpiredToken.Error() == lower {
		errCode = types.ErrJWTTokenExpired
	} else {
		errCode = types.ErrJWTCommonErr
	}
	types.RenderJSON(c, errCode, err, nil)
}

func loginResponse(c *gin.Context, code int, message string, time time.Time) {
	resp := coordinatorType.LoginSchema{
		Time:  time,
		Token: message,
	}
	types.RenderJSON(c, types.Success, nil, resp)
}
