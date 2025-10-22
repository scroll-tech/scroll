package proxy

import (
	"fmt"

	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/controller/api"
	"scroll-tech/coordinator/internal/logic/auth"
	"scroll-tech/coordinator/internal/logic/verifier"
	"scroll-tech/coordinator/internal/types"
)

// AuthController is login API
type AuthController struct {
	apiLogin  *api.AuthController
	clients   Clients
	proverMgr *ProverManager
}

const upstreamConnTimeout = time.Second * 2
const LoginParamCache = "login_param"
const ProverTypesKey = "prover_types"
const SignatureKey = "prover_signature"

// NewAuthController returns an LoginController instance
func NewAuthController(cfg *config.ProxyConfig, clients Clients, proverMgr *ProverManager) *AuthController {

	// use a dummy Verifier to create login logic (we do not use any information in verifier)
	dummyVf := verifier.Verifier{
		OpenVMVkMap: make(map[string]struct{}),
	}
	loginLogic := auth.NewLoginLogicWithSimpleDeduplicator(cfg.ProxyManager.Verifier, &dummyVf)

	authController := &AuthController{
		apiLogin:  api.NewAuthControllerWithLogic(loginLogic),
		clients:   clients,
		proverMgr: proverMgr,
	}

	return authController
}

// Login extended the Login hander in api controller
func (a *AuthController) Login(c *gin.Context) (interface{}, error) {

	loginRes, err := a.apiLogin.Login(c)
	if err != nil {
		return nil, err
	}
	loginParam := loginRes.(types.LoginParameterWithHardForkName)

	if loginParam.LoginParameter.Message.ProverProviderType == types.ProverProviderTypeProxy {
		return nil, fmt.Errorf("proxy do not support recursive login")
	}

	session := a.proverMgr.GetOrCreate(loginParam.PublicKey, loginParam.Message.ProverName)
	log.Debug("start handling login", "cli", loginParam.Message.ProverName)

	for _, cli := range a.clients {

		go func(cli Client) {
			if err := session.ProxyLogin(c, cli, &loginParam.LoginParameter); err != nil {
				log.Error("proxy login failed during token cache update",
					"userKey", loginParam.PublicKey,
					"upstream", cli.Name(),
					"error", err)
			}
		}(cli)
	}

	return loginParam.LoginParameter, nil
}

// PayloadFunc returns jwt.MapClaims with {public key, prover name}.
func (a *AuthController) PayloadFunc(data interface{}) jwt.MapClaims {
	v, ok := data.(types.LoginParameter)
	if !ok {
		return jwt.MapClaims{}
	}

	return jwt.MapClaims{
		types.PublicKey:             v.PublicKey,
		types.ProverName:            v.Message.ProverName,
		types.ProverVersion:         v.Message.ProverVersion,
		types.ProverProviderTypeKey: v.Message.ProverProviderType,
		SignatureKey:                v.Signature,
		ProverTypesKey:              v.Message.ProverTypes,
	}
}

// IdentityHandler replies to client for /login
func (a *AuthController) IdentityHandler(c *gin.Context) interface{} {
	claims := jwt.ExtractClaims(c)
	loginParam := &types.LoginParameter{}

	if proverName, ok := claims[types.ProverName]; ok {
		loginParam.Message.ProverName, _ = proverName.(string)
	}

	if proverVersion, ok := claims[types.ProverVersion]; ok {
		loginParam.Message.ProverVersion, _ = proverVersion.(string)
	}

	if providerType, ok := claims[types.ProverProviderTypeKey]; ok {
		num, _ := providerType.(float64)
		loginParam.Message.ProverProviderType = types.ProverProviderType(num)
	}

	if signature, ok := claims[SignatureKey]; ok {
		loginParam.Signature, _ = signature.(string)
	}

	if proverTypes, ok := claims[ProverTypesKey]; ok {
		arr, _ := proverTypes.([]any)
		for _, elm := range arr {
			num, _ := elm.(float64)
			loginParam.Message.ProverTypes = append(loginParam.Message.ProverTypes, types.ProverType(num))
		}
	}

	if publicKey, ok := claims[types.PublicKey]; ok {
		loginParam.PublicKey, _ = publicKey.(string)
	}

	if loginParam.PublicKey != "" {

		c.Set(LoginParamCache, loginParam)
		return loginParam.PublicKey
	}

	return nil
}
