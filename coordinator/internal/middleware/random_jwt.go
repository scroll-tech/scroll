package middleware

import (
	"github.com/gin-gonic/gin"
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/coordinator/internal/config"
)

// RandomMiddleware jwt random middleware
func RandomMiddleware(conf *config.Config) *jwt.GinJWTMiddleware {
	jwtMiddleware, err := jwt.New(&jwt.GinJWTMiddleware{
		Authenticator: func(c *gin.Context) (interface{}, error) {
			return nil, nil
		},
		Unauthorized:  unauthorized,
		Key:           []byte(conf.Auth.Secret),
		Timeout:       time.Second * time.Duration(conf.Auth.RandomExpireDuration),
		TokenLookup:   "header: Authorization, query: token, cookie: jwt",
		TokenHeadName: "Bearer",
		TimeFunc:      time.Now,
		LoginResponse: loginResponse,
	})

	if err != nil {
		log.Crit("new jwt middleware panic", "error", err)
	}

	if errInit := jwtMiddleware.MiddlewareInit(); errInit != nil {
		log.Crit("init jwt middleware panic", "error", errInit)
	}

	return jwtMiddleware
}
