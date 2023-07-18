package controller

import (
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"scroll-tech/prover-stats-api/internal/types"
)

type AuthController struct {
}

func NewAuthController() *AuthController {
	return &AuthController{}
}

// Login godoc
// @Summary    	 login with prover public key
// @Description  login with prover public key
// @Tags         prover_task
// @Accept       plain
// @Produce      plain
// @Param        pubkey   query  string  true  "prover public key"
// @Success      200  {array}   types.LoginSchema
// @Router       /api/prover_task/v1/request_token [get]
func (a *AuthController) Login(c *gin.Context) (interface{}, error) {
	var login types.LoginParameter
	if err := c.ShouldBindQuery(&login); err != nil {
		return "", fmt.Errorf("missing the public_key, err:%w", err)
	}

	// TODO check public key is exist

	if a.checkValidPublicKey() {
		return types.LoginParameter{PublicKey: login.PublicKey}, nil
	}

	return nil, errors.New("incorrect public_key")
}

func (a *AuthController) checkValidPublicKey() bool {
	return true
}

func (a *AuthController) LoginResponse(c *gin.Context, code int, message string, time time.Time) {
	resp := types.LoginSchema{
		Time:  time,
		Token: message,
	}
	types.RenderJson(c, code, nil, resp)
}
