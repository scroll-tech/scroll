package middleware

import (
	"scroll-tech/coordinator/internal/types"
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/controller/api"
)

// LoginMiddleware jwt auth middleware
func LoginMiddleware(conf *config.Config) *jwt.GinJWTMiddleware {
	jwtMiddleware, err := jwt.New(&jwt.GinJWTMiddleware{
		PayloadFunc:     api.Auth.PayloadFunc,
		IdentityHandler: api.Auth.IdentityHandler,
		IdentityKey:     types.PublicKey,
		Key:             []byte(conf.Auth.Secret),
		Timeout:         time.Second * time.Duration(conf.Auth.LoginExpireDuration),
		Authenticator:   api.Auth.Login,
		Unauthorized:    unauthorized,
		TokenLookup:     "header: Authorization, query: token, cookie: jwt",
		TokenHeadName:   "Bearer",
		TimeFunc:        time.Now,
		LoginResponse:   loginResponse,
	})

	if err != nil {
		log.Crit("new jwt middleware panic", "error", err)
	}

	if errInit := jwtMiddleware.MiddlewareInit(); errInit != nil {
		log.Crit("init jwt middleware panic", "error", errInit)
	}

	return jwtMiddleware
}
