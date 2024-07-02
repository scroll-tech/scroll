package api

import (
	"errors"
	"fmt"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/auth"
	"scroll-tech/coordinator/internal/logic/verifier"
	"scroll-tech/coordinator/internal/types"
)

// AuthController is login API
type AuthController struct {
	loginLogic *auth.LoginLogic
}

// NewAuthController returns an LoginController instance
func NewAuthController(db *gorm.DB, cfg *config.Config, vf *verifier.Verifier) *AuthController {
	return &AuthController{
		loginLogic: auth.NewLoginLogic(db, cfg, vf),
	}
}

// Login the api controller for login
func (a *AuthController) Login(c *gin.Context) (interface{}, error) {
	var login types.LoginParameter
	if err := c.ShouldBind(&login); err != nil {
		return "", fmt.Errorf("missing the public_key, err:%w", err)
	}

	// check login parameter's token is equal to bearer token, the Authorization must be existed
	// if not exist, the jwt token will intercept it
	brearToken := c.GetHeader("Authorization")
	if brearToken != "Bearer "+login.Message.Challenge {
		return "", errors.New("check challenge failure for the not equal challenge string")
	}

	if err := a.loginLogic.Check(&login); err != nil {
		return "", fmt.Errorf("check the login parameter failure: %w", err)
	}

	// check the challenge is used, if used, return failure
	if err := a.loginLogic.InsertChallengeString(c, login.Message.Challenge); err != nil {
		return "", fmt.Errorf("login insert challenge string failure:%w", err)
	}
	return login, nil
}

// PayloadFunc returns jwt.MapClaims with {public key, prover name}.
func (a *AuthController) PayloadFunc(data interface{}) jwt.MapClaims {
	v, ok := data.(types.LoginParameter)
	if !ok {
		return jwt.MapClaims{}
	}

	return jwt.MapClaims{
		types.PublicKey:     v.PublicKey,
		types.ProverName:    v.Message.ProverName,
		types.ProverVersion: v.Message.ProverVersion,
		types.VKs:           v.Message.VKs,
	}
}

// IdentityHandler replies to client for /login
func (a *AuthController) IdentityHandler(c *gin.Context) interface{} {
	claims := jwt.ExtractClaims(c)
	if proverName, ok := claims[types.ProverName]; ok {
		c.Set(types.ProverName, proverName)
	}

	if publicKey, ok := claims[types.PublicKey]; ok {
		c.Set(types.PublicKey, publicKey)
	}

	if proverVersion, ok := claims[types.ProverVersion]; ok {
		c.Set(types.ProverVersion, proverVersion)
	}

	if vks, ok := claims[types.VKs]; ok {
		c.Set(types.VKs, vks)
	}
	return nil
}
