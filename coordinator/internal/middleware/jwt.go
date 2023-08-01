package middleware

import (
	"errors"
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"

	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/controller/api"
	"scroll-tech/coordinator/internal/types"
)

const (
	// IdentityKey is auth key
	IdentityKey = "public_key"
	// ProverName prover name
	ProverName = "prover_name"
)

// AuthMiddleware jwt auth middleware
func AuthMiddleware(conf *config.Config) *jwt.GinJWTMiddleware {
	jwtMiddleware, err := jwt.New(&jwt.GinJWTMiddleware{
		PayloadFunc:     PayloadFunc,
		IdentityHandler: IdentityHandler,
		IdentityKey:     IdentityKey,
		Key:             []byte(conf.Auth.Secret),
		Timeout:         time.Second * time.Duration(conf.Auth.TokenExpireDuration),
		Authenticator:   api.Auth.Login,
		Unauthorized:    Unauthorized,
		TokenLookup:     "header: Authorization, query: token, cookie: jwt",
		TokenHeadName:   "Bearer",
		TimeFunc:        time.Now,
		LoginResponse:   api.Auth.LoginResponse,
		Authorizator:    api.Auth.Authorizator,
	})

	if err != nil {
		log.Crit("new jwt middleware panic", "error", err)
	}

	if errInit := jwtMiddleware.MiddlewareInit(); errInit != nil {
		log.Crit("init jwt middleware panic", "error", errInit)
	}

	return jwtMiddleware
}

// Unauthorized response Unauthorized error message to client
func Unauthorized(c *gin.Context, code int, message string) {
	err := errors.New(message)
	types.RenderJSON(c, code, err, nil)
}

// PayloadFunc returns jwt.MapClaims with {public key, prover name}.
func PayloadFunc(data interface{}) jwt.MapClaims {
	if v, ok := data.(types.LoginParameter); ok {
		return jwt.MapClaims{
			IdentityKey: v.PublicKey,
			ProverName:  v.ProverName,
		}
	}
	return jwt.MapClaims{}
}

// IdentityHandler replies to client for /login
func IdentityHandler(c *gin.Context) interface{} {
	claims := jwt.ExtractClaims(c)
	return &types.LoginParameter{
		PublicKey:  claims[IdentityKey].(string),
		ProverName: claims[ProverName].(string),
	}
}
