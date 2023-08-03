package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/gin-gonic/gin"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/coordinator/internal/config"
)

// ChallengeMiddleware jwt challenge middleware
func ChallengeMiddleware(conf *config.Config) *jwt.GinJWTMiddleware {
	jwtMiddleware, err := jwt.New(&jwt.GinJWTMiddleware{
		Authenticator: func(c *gin.Context) (interface{}, error) {
			return nil, nil
		},
		PayloadFunc: func(data interface{}) jwt.MapClaims {
			b := make([]byte, 32)
			_, err := rand.Read(b)
			if err != nil {
				return jwt.MapClaims{}
			}
			return jwt.MapClaims{
				"random": base64.URLEncoding.EncodeToString(b),
			}
		},
		Unauthorized:  unauthorized,
		Key:           []byte(conf.Auth.Secret),
		Timeout:       time.Second * time.Duration(conf.Auth.ChallengeExpireDurationSec),
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
