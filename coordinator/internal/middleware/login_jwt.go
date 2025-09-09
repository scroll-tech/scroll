package middleware

import (
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/controller/api"
	"scroll-tech/coordinator/internal/controller/proxy"
	"scroll-tech/coordinator/internal/types"
)

func nonIdendityAuthorizator(data interface{}, _ *gin.Context) bool {
	if data == nil {
		return false
	}
	return true
}

// LoginMiddleware jwt auth middleware
func LoginMiddleware(auth *config.Auth) *jwt.GinJWTMiddleware {
	jwtMiddleware, err := jwt.New(&jwt.GinJWTMiddleware{
		PayloadFunc:     api.Auth.PayloadFunc,
		IdentityHandler: api.Auth.IdentityHandler,
		IdentityKey:     types.PublicKey,
		Key:             []byte(auth.Secret),
		Timeout:         time.Second * time.Duration(auth.LoginExpireDurationSec),
		Authenticator:   api.Auth.Login,
		Authorizator:    nonIdendityAuthorizator,
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

// ProxyLoginMiddleware jwt auth middleware for proxy login
func ProxyLoginMiddleware(auth *config.Auth) *jwt.GinJWTMiddleware {
	jwtMiddleware, err := jwt.New(&jwt.GinJWTMiddleware{
		PayloadFunc:     proxy.Auth.PayloadFunc,
		IdentityHandler: proxy.Auth.IdentityHandler,
		IdentityKey:     types.PublicKey,
		Key:             []byte(auth.Secret),
		Timeout:         time.Second * time.Duration(auth.LoginExpireDurationSec),
		Authenticator:   proxy.Auth.Login,
		Authorizator:    nonIdendityAuthorizator,
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
