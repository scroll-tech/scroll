package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

const TokenExpireDuration = time.Minute * 10

var (
	Secret    string
	skipPaths = []string{"/api/v1/prover_task/request_token"}
)

type ApiClaims struct {
	jwt.StandardClaims
}

func GenToken() (string, error) {
	c := ApiClaims{
		jwt.StandardClaims{
			ExpiresAt: time.Now().Add(TokenExpireDuration).Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString([]byte(Secret))
}

func ParseToken(tokenStr string) (*ApiClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &ApiClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(Secret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*ApiClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}

func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, path := range skipPaths {
			if path == c.FullPath() {
				c.Next()
				return
			}
		}
		tokenString := c.Request.Header.Get("Authorization")
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Null token"})
			return
		}

		_, err := ParseToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("Unauthorized: %v", err)})
			return
		}
		c.Next()
	}
}
