package proxy

import (
	"fmt"
	"sync"

	"context"
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
	apiLogin       *api.AuthController
	clients        Clients
	userTokenCache *UserTokenCache
}

type TokenUpdate struct {
	PublicKey      string
	Upstream       string
	Phase          uint
	LoginParam     types.LoginParameter
	CompleteNotify chan<- *types.LoginSchema
}

type UpstreamTokens struct {
	LoginData      map[string]*types.LoginSchema
	LoginPhase     uint
	NextLoginPhase uint
}

type UserTokenCache struct {
	sync.RWMutex
	data             map[string]UpstreamTokens
	tokenCacheUpdate chan<- *TokenUpdate
}

func newUserTokens() UpstreamTokens {
	return UpstreamTokens{
		LoginData: make(map[string]*types.LoginSchema),
	}
}

func newUserCache(tokenCacheUpdate chan<- *TokenUpdate) *UserTokenCache {
	return &UserTokenCache{
		data:             make(map[string]UpstreamTokens),
		tokenCacheUpdate: tokenCacheUpdate,
	}
}

// get retrieves UpstreamTokens for a given user key, returns empty if still not exists
func (c *UserTokenCache) Get(userKey string) *UpstreamTokens {
	c.RLock()
	defer c.RUnlock()

	tokens, exists := c.data[userKey]
	if !exists {
		return nil
	}

	return &tokens
}

// prepare for a total update via Login request
func (c *UserTokenCache) updatePrepare(userKey string) UpstreamTokens {
	c.Lock()
	defer c.Unlock()

	if _, exists := c.data[userKey]; !exists {
		log.Info("initializing user token cache", "userKey", userKey)
		c.data[userKey] = newUserTokens()
	}
	updated := c.data[userKey]
	updated.NextLoginPhase = updated.LoginPhase + 1
	c.data[userKey] = updated
	return updated
}

// partialSet updates a single entry in upstreamTokens for a given user
func (c *UserTokenCache) partialSet(userKey string, upstreamName string, loginSchema *types.LoginSchema, phase uint) {
	c.Lock()
	defer c.Unlock()

	// Get existing tokens or create new map
	tokens, exists := c.data[userKey]
	if exists && tokens.NextLoginPhase == phase {
		// Update the specific upstream entry
		tokens.LoginData[upstreamName] = loginSchema
	}
}

// LoginParameterWithHardForkName constructs new payload for login
type LoginParameterWithUpstreamTokens struct {
	*types.LoginParameter
	Tokens UpstreamTokens
}

const upstreamConnTimeout = time.Second * 2
const expireTolerant = 10 * time.Minute
const LoginParamCache = "login_param"
const ProverTypesKey = "prover_types"
const SignatureKey = "prover_signature"

// NewAuthController returns an LoginController instance
func NewAuthController(cfg *config.ProxyConfig, clients Clients, vf *verifier.Verifier) *AuthController {

	loginLogic := auth.NewLoginLogicWithSimpleDEduplicator(cfg.ProxyManager.Verifier, vf)

	// Create the token cache update channel
	tokenCacheUpdateChan := make(chan *TokenUpdate)

	authController := &AuthController{
		apiLogin:       api.NewAuthControllerWithLogic(loginLogic),
		clients:        clients,
		userTokenCache: newUserCache(tokenCacheUpdateChan),
	}

	// Launch token cache manager in a separate goroutine
	go authController.toeknCacheManager(tokenCacheUpdateChan)

	return authController
}

func (a *AuthController) TokenCache() *UserTokenCache { return a.userTokenCache }

func (a *AuthController) doUpdateRequest(ctx context.Context, req *TokenUpdate) (ret *types.LoginSchema) {
	if req.CompleteNotify != nil {
		defer func(ctx context.Context) {
			select {
			case <-ctx.Done():
			case req.CompleteNotify <- ret:
			}

		}(ctx)
	}

	cli := a.clients[req.Upstream]
	if cli := cli.Client(ctx); cli != nil {
		var err error
		if ret, err = cli.ProxyLogin(ctx, &req.LoginParam); err == nil {
			a.userTokenCache.partialSet(req.PublicKey, req.Upstream, ret, req.Phase)
		} else {
			log.Error("proxy login failed during token cache update",
				"userKey", req.PublicKey,
				"upstream", req.Upstream,
				"phase", req.Phase,
				"error", err)
		}
	}
	return

}

func (a *AuthController) toeknCacheManager(request <-chan *TokenUpdate) {

	ctx := context.TODO()
	var managerStatusLock sync.Mutex
	managerStatus := make(map[string]map[string]uint)

	for {
		req, ok := <-request
		if !ok {
			return
		}

		// ensure the manager request is not outdated
		tokens := a.userTokenCache.Get(req.PublicKey)
		if tokens == nil {
			// Highly not possible, if raise, the reason is unknown, just log the Error
			continue
		}
		phase := tokens.NextLoginPhase
		if req.Phase < phase {
			// drop the out-dated request
			continue
		}

		// ensure only one login request is launched for the same phase
		managerStatusLock.Lock()
		stat, ok := managerStatus[req.Upstream]
		if !ok {
			managerStatus[req.Upstream] = make(map[string]uint)
			stat = managerStatus[req.Upstream]
		}
		if phase, running := stat[req.PublicKey]; running && phase >= req.Phase {
			managerStatusLock.Unlock()
			continue
		} else {
			stat[req.PublicKey] = req.Phase
		}
		managerStatusLock.Unlock()

		go a.doUpdateRequest(ctx, req)

	}

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

	tokens := a.userTokenCache.updatePrepare(loginParam.PublicKey)
	notifies := make([]chan *types.LoginSchema, len(a.clients))

	for n := range a.clients {

		// Check if we have a valid cached token that hasn't expired
		if knownEntry, existed := tokens.LoginData[n]; existed {
			timeRemaining := time.Until(knownEntry.Time)
			if timeRemaining > expireTolerant {
				// Token is still valid enouth, continue to next client
				continue
			}
		}

		notify := make(chan *types.LoginSchema)
		notifies = append(notifies, notify)
		request := TokenUpdate{
			PublicKey:      loginParam.PublicKey,
			Upstream:       n,
			Phase:          tokens.NextLoginPhase,
			LoginParam:     loginParam.LoginParameter,
			CompleteNotify: notify,
		}
		defer close(notify)
		select {
		case <-c.Done():
		case a.userTokenCache.tokenCacheUpdate <- &request:
		}

	}

	// collect all request's compeletions
	for _, chn := range notifies {
		select {
		case <-c.Done():
		case <-chn:
		}
	}

	return LoginParameterWithUpstreamTokens{
		LoginParameter: &loginParam.LoginParameter,
		Tokens:         tokens,
	}, nil
}

// PayloadFunc returns jwt.MapClaims with {public key, prover name}.
func (a *AuthController) PayloadFunc(data interface{}) jwt.MapClaims {
	v, ok := data.(LoginParameterWithUpstreamTokens)
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
		// ensure tokenCache
		a.userTokenCache.RLock()
		_, exists := a.userTokenCache.data[loginParam.PublicKey]
		if !exists {
			a.userTokenCache.RUnlock()
			a.userTokenCache.Lock()
			if _, exists := a.userTokenCache.data[loginParam.PublicKey]; !exists {
				log.Info("creating token cache for user after proxy restart",
					"publicKey", loginParam.PublicKey,
					"proverName", loginParam.Message.ProverName,
					"reason", "prover using JWT token from before proxy restart")
				a.userTokenCache.data[loginParam.PublicKey] = newUserTokens()
			}
			a.userTokenCache.Unlock()
		} else {
			a.userTokenCache.RUnlock()
		}

		c.Set(LoginParamCache, loginParam)
		return loginParam.PublicKey
	}

	return nil
}
