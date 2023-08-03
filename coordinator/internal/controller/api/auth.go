package api

import (
	"fmt"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/logic/auth"
	"scroll-tech/coordinator/internal/types"
)

// AuthController is login API
type AuthController struct {
	loginLogic *auth.LoginLogic
}

// NewAuthController returns an LoginController instance
func NewAuthController(db *gorm.DB) *AuthController {
	return &AuthController{
		loginLogic: auth.NewLoginLogic(db),
	}
}

// Login the api controller for login
func (a *AuthController) Login(c *gin.Context) (interface{}, error) {
	var login types.LoginParameter
	if err := c.ShouldBind(&login); err != nil {
		return "", fmt.Errorf("missing the public_key, err:%w", err)
	}
	// check the random is used, if used, return failure
	if err := a.loginLogic.InsertRandomString(c, login.Signature); err != nil {
		return "", fmt.Errorf("login insert random string failure:%w", err)
	}
	return login, nil
}

// PayloadFunc returns jwt.MapClaims with {public key, prover name}.
func (a *AuthController) PayloadFunc(data interface{}) jwt.MapClaims {
	v, ok := data.(types.LoginParameter)
	if !ok {
		return jwt.MapClaims{}
	}

	// recover the public key
	authMsg := message.AuthMsg{
		Identity: &message.Identity{
			Random:     v.Message.Random,
			ProverName: v.Message.ProverName,
		},
		Signature: v.Signature,
	}

	publicKey, err := authMsg.PublicKey()
	if err != nil {
		return jwt.MapClaims{}
	}

	return jwt.MapClaims{
		types.PublicKey:  publicKey,
		types.ProverName: v.Message.ProverName,
	}
}

// IdentityHandler replies to client for /login
func (a *AuthController) IdentityHandler(c *gin.Context) interface{} {
	claims := jwt.ExtractClaims(c)
	c.Set(types.ProverName, claims[types.ProverName])
	return claims[types.PublicKey]
}
