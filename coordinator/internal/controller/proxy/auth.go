package proxy

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/controller/api"
	"scroll-tech/coordinator/internal/logic/auth"
	"scroll-tech/coordinator/internal/logic/verifier"
	"scroll-tech/coordinator/internal/types"
)

// AuthController is login API
type AuthController struct {
	*api.AuthController
	clients Clients
}

// NewAuthController returns an LoginController instance
func NewAuthController(cfg *config.ProxyConfig, clients Clients, vf *verifier.Verifier) *AuthController {

	loginLogic := auth.NewLoginLogicWithSimpleDEduplicator(cfg.ProxyManager.Verifier, vf)
	auth := api.NewAuthControllerWithLogic(loginLogic)
	return &AuthController{
		AuthController: auth,
		clients:        clients,
	}
}

// Login extended the Login hander in api controller
func (a *AuthController) Login(c *gin.Context) (interface{}, error) {

	ret, err := a.AuthController.Login(c)
	if err != nil {
		return nil, err
	}
	loginParam := ret.(types.LoginParameterWithHardForkName)
	// band recursive proxy now ...
	if loginParam.Message.ProverProviderType == types.ProverProviderTypeProxy {
		return nil, fmt.Errorf("do not allow recursive proxy for login %v", loginParam.Message)
	}

	return loginParam, nil
}
