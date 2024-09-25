package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/scroll-tech/go-ethereum/log"
	"scroll-tech/coordinator/internal/config"
)

func ChallengeMiddleware(conf *config.Config) *jwt.GinJWTMiddleware {
	jwtMiddleware, err := jwt.New(&jwt.GinJWTMiddleware{
		Authenticator: func(c *gin.Context) (interface{}, error) {
			log.Info("Attempting to authenticate user")
			return nil, errors.New("authentication failed: no logic provided")
		},
		PayloadFunc: func(data interface{}) jwt.MapClaims {
			b := make([]byte, 32)
			_, err := rand.Read(b)
			if err != nil {
				log.Error("Error generating random bytes for JWT payload", "error", err)
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
		log.Crit("Failed to create new JWT middleware", "error", err)
		panic(err)
	}

	if errInit := jwtMiddleware.MiddlewareInit(); errInit != nil {
		log.Crit("Failed to initialize JWT middleware", "error", errInit)
		panic(errInit)
	}

	return jwtMiddleware
}
