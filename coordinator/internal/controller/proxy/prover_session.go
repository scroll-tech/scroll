package proxy

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/scroll-tech/go-ethereum/log"

	ctypes "scroll-tech/common/types"
	"scroll-tech/coordinator/internal/types"
)

type ProverManager struct {
	sync.RWMutex
	data               map[string]*proverSession
	willDeprecatedData map[string]*proverSession
	sizeLimit          int
	persistent         *proverDataPersist
}

func NewProverManager(size int) *ProverManager {
	return &ProverManager{
		data:               make(map[string]*proverSession),
		willDeprecatedData: make(map[string]*proverSession),
		sizeLimit:          size,
	}
}

func NewProverManagerWithPersistent(size int, db *gorm.DB) *ProverManager {
	return &ProverManager{
		data:               make(map[string]*proverSession),
		willDeprecatedData: make(map[string]*proverSession),
		sizeLimit:          size,
		persistent:         NewProverDataPersist(db),
	}
}

// get retrieves ProverSession for a given user key, returns empty if still not exists
func (m *ProverManager) Get(userKey string) (ret *proverSession) {
	defer func() {
		r := ret
		if r == nil {
			var err error
			r, err = m.persistent.Get(userKey)
			if err != nil {
				log.Error("Get persistent layer for prover tokens fail", "error", err)
			} else if r != nil {
				r.persistent = m.persistent
			}
		}

		if r != nil {
			m.Lock()
			m.data[userKey] = r
			m.Unlock()
		}
	}()

	m.RLock()
	defer m.RUnlock()
	if r, existed := m.data[userKey]; existed {
		return r
	} else {
		return m.willDeprecatedData[userKey]
	}
}

func (m *ProverManager) GetOrCreate(userKey, cliName string) *proverSession {

	if ret := m.Get(userKey); ret != nil {
		return ret
	}

	m.Lock()
	defer m.Unlock()

	ret := &proverSession{
		proverToken: make(map[string]loginToken),
		CliName:     cliName,
		persistent:  m.persistent,
	}

	if len(m.data) >= m.sizeLimit {
		m.willDeprecatedData = m.data
		m.data = make(map[string]*proverSession)
	}

	m.data[userKey] = ret
	return ret
}

type loginToken struct {
	*types.LoginSchema
	phase uint
}

// Client wraps an http client with a preset host for coordinator API calls
type proverSession struct {
	CliName    string
	persistent *proverDataPersist

	sync.RWMutex
	proverToken   map[string]loginToken
	completionCtx context.Context
}

func (c *proverSession) maintainLogin(ctx context.Context, cliMgr Client, up string, param *types.LoginParameter, phase uint) (result loginToken, nerr error) {
	c.Lock()
	curPhase := c.proverToken[up].phase
	if c.completionCtx != nil {
		waitctx := c.completionCtx
		c.Unlock()
		select {
		case <-waitctx.Done():
			return c.maintainLogin(ctx, cliMgr, up, param, phase)
		case <-ctx.Done():
			nerr = fmt.Errorf("ctx fail")
			return
		}
	}

	if phase < curPhase {
		// outdate login phase, give up
		log.Debug("drop outdated proxy login attemp", "upstream", up, "cli", param.Message.ProverName, "phase", phase, "now", curPhase)
		defer c.Unlock()
		return c.proverToken[up], nil
	}

	// occupy the update slot
	completeCtx, cf := context.WithCancel(ctx)
	defer cf()
	c.completionCtx = completeCtx
	defer func() {
		c.Lock()
		c.completionCtx = nil
		if result.LoginSchema != nil {
			c.proverToken[up] = result
			log.Info("maintain login status", "upstream", up, "cli", param.Message.ProverName, "phase", curPhase+1)
		}
		c.Unlock()
		if nerr != nil {
			log.Error("maintain login fail", "error", nerr, "upstream", up, "cli", param.Message.ProverName, "phase", curPhase)
		}
	}()
	c.Unlock()

	log.Debug("start proxy login process", "upstream", up, "cli", param.Message.ProverName)
	c.CliName = param.Message.ProverName

	cli := cliMgr.Client(ctx)
	if cli == nil {
		nerr = fmt.Errorf("get upstream cli fail")
		return
	}

	resp, err := cli.ProxyLogin(ctx, param)
	if err != nil {
		nerr = fmt.Errorf("proxylogin fail: %v", err)
		return
	}

	if resp.ErrCode == ctypes.ErrJWTTokenExpired {
		cliMgr.Reset(cli)
		cli = cliMgr.Client(ctx)
		if cli == nil {
			nerr = fmt.Errorf("get upstream cli fail (secondary try)")
			return
		}

		// like SDK, we would try one more time if the upstream token is expired
		resp, err = cli.ProxyLogin(ctx, param)
		if err != nil {
			nerr = fmt.Errorf("proxylogin fail: %v", err)
			return
		}
	}

	if resp.ErrCode != 0 {
		nerr = fmt.Errorf("upstream fail: %d (%s)", resp.ErrCode, resp.ErrMsg)
		return
	}

	var loginResult loginSchema
	if err := resp.DecodeData(&loginResult); err != nil {
		nerr = err
		return
	}

	log.Debug("Proxy login done", "upstream", up, "cli", param.Message.ProverName)
	result = loginToken{
		LoginSchema: &types.LoginSchema{
			Token: loginResult.Token,
		},
		phase: curPhase + 1,
	}
	return
}

const expireTolerant = 10 * time.Minute

// ProxyLogin makes a POST request to /v1/proxy_login with LoginParameter
func (c *proverSession) ProxyLogin(ctx context.Context, cli Client, up string, param *types.LoginParameter) error {
	c.RLock()
	existedToken := c.proverToken[up]
	c.RUnlock()

	newtoken, err := c.maintainLogin(ctx, cli, up, param, math.MaxUint)
	if newtoken.phase > existedToken.phase {
		if err := c.persistent.Update(param.PublicKey, up, newtoken.LoginSchema); err != nil {
			log.Error("Update persistent layer for prover tokens fail", "error", err)
		}
	}

	return err
}

// GetTask makes a POST request to /v1/get_task with GetTaskParameter
func (c *proverSession) GetTask(ctx context.Context, param *types.GetTaskParameter, cliMgr Client, up string) (*ctypes.Response, error) {
	c.RLock()
	token := c.proverToken[up]
	c.RUnlock()

	cli := cliMgr.Client(ctx)
	if cli == nil {
		return nil, fmt.Errorf("get upstream cli fail")
	}

	if token.LoginSchema != nil {
		resp, err := cli.GetTask(ctx, param, token.Token)
		if err != nil {
			return nil, err
		}
		if resp.ErrCode != ctypes.ErrJWTTokenExpired {
			return resp, nil
		}
	}

	// like SDK, we would try one more time if the upstream token is expired
	// get param from ctx
	loginParam, ok := ctx.Value(LoginParamCache).(*types.LoginParameter)
	if !ok {
		return nil, fmt.Errorf("Unexpected error, no loginparam ctx value")
	}

	newToken, err := c.maintainLogin(ctx, cliMgr, up, loginParam, token.phase)
	if err != nil {
		return nil, fmt.Errorf("update prover token fail: %v", err)
	}

	return cli.GetTask(ctx, param, newToken.Token)

}

// SubmitProof makes a POST request to /v1/submit_proof with SubmitProofParameter
func (c *proverSession) SubmitProof(ctx context.Context, param *types.SubmitProofParameter, cliMgr Client, up string) (*ctypes.Response, error) {
	c.RLock()
	token := c.proverToken[up]
	c.RUnlock()

	cli := cliMgr.Client(ctx)
	if cli == nil {
		return nil, fmt.Errorf("get upstream cli fail")
	}

	if token.LoginSchema != nil {
		resp, err := cli.SubmitProof(ctx, param, token.Token)
		if err != nil {
			return nil, err
		}
		if resp.ErrCode != ctypes.ErrJWTTokenExpired {
			return resp, nil
		}
	}

	// like SDK, we would try one more time if the upstream token is expired
	// get param from ctx
	loginParam, ok := ctx.Value(LoginParamCache).(*types.LoginParameter)
	if !ok {
		return nil, fmt.Errorf("Unexpected error, no loginparam ctx value")
	}

	newToken, err := c.maintainLogin(ctx, cliMgr, up, loginParam, token.phase)
	if err != nil {
		return nil, fmt.Errorf("update prover token fail: %V", err)
	}

	return cli.SubmitProof(ctx, param, newToken.Token)
}
