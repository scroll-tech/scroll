package proxy

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	ctypes "scroll-tech/common/types"
	"scroll-tech/coordinator/internal/types"
)

type ProverManager struct {
	sync.RWMutex
	data map[string]*proverSession
}

func NewProverManager() *ProverManager {
	return &ProverManager{
		data: make(map[string]*proverSession),
	}
}

// get retrieves ProverSession for a given user key, returns empty if still not exists
func (m *ProverManager) Get(userKey string) *proverSession {
	m.RLock()
	defer m.RUnlock()

	return m.data[userKey]
}

func (m *ProverManager) GetOrCreate(userKey string) *proverSession {
	m.Lock()
	defer m.Unlock()

	if ret, ok := m.data[userKey]; ok {
		return ret
	}

	ret := &proverSession{
		proverToken: make(map[string]loginToken),
	}

	m.data[userKey] = ret
	return ret
}

func (m *ProverManager) Set(userKey string, session *proverSession) {
	m.Lock()
	defer m.Unlock()

	m.data[userKey] = session
}

type loginToken struct {
	*types.LoginSchema
	phase uint
}

// Client wraps an http client with a preset host for coordinator API calls
type proverSession struct {
	sync.RWMutex
	proverToken   map[string]loginToken
	completionCtx context.Context
}

func (c *proverSession) maintainLogin(ctx context.Context, cliMgr Client, up string, param *types.LoginParameter, phase uint) (*types.LoginSchema, error) {
	c.Lock()
	curPhase := c.proverToken[up].phase
	if c.completionCtx != nil {
		waitctx := c.completionCtx
		c.Unlock()
		select {
		case <-waitctx.Done():
			return c.maintainLogin(ctx, cliMgr, up, param, phase)
		case <-ctx.Done():
			return nil, fmt.Errorf("ctx fail")
		}
	}

	if phase < curPhase {
		// outdate login phase, give up
		defer c.Unlock()
		return c.proverToken[up].LoginSchema, nil
	}

	// occupy the update slot
	completeCtx, cf := context.WithCancel(ctx)
	defer cf()
	c.completionCtx = completeCtx
	c.Unlock()

	cli := cliMgr.Client(ctx)
	if cli == nil {
		return nil, fmt.Errorf("get upstream cli fail")
	}

	resp, err := cli.ProxyLogin(ctx, param)
	if err != nil {
		return nil, fmt.Errorf("proxylogin fail: %v", err)
	}

	if resp.ErrCode == ctypes.ErrJWTTokenExpired {
		cliMgr.Reset(cli)
		cli = cliMgr.Client(ctx)
		if cli == nil {
			return nil, fmt.Errorf("get upstream cli fail (secondary try)")
		}

		// like SDK, we would try one more time if the upstream token is expired
		resp, err = cli.ProxyLogin(ctx, param)
		if err != nil {
			return nil, fmt.Errorf("proxylogin fail: %v", err)
		}
	}

	if resp.ErrCode != 0 {
		return nil, fmt.Errorf("upstream fail: %d (%s)", resp.ErrCode, resp.ErrMsg)
	}

	var loginResult loginSchema
	if err := resp.DecodeData(&loginResult); err != nil {
		return nil, err
	}

	c.Lock()
	defer c.Unlock()

	c.proverToken[up] = loginToken{
		LoginSchema: &types.LoginSchema{
			Token: loginResult.Token,
		},
		phase: curPhase + 1,
	}
	c.completionCtx = nil

	return c.proverToken[up].LoginSchema, nil
}

const expireTolerant = 10 * time.Minute

// ProxyLogin makes a POST request to /v1/proxy_login with LoginParameter
func (c *proverSession) ProxyLogin(ctx context.Context, cli Client, up string, param *types.LoginParameter) error {
	c.RLock()
	existedToken := c.proverToken[up].LoginSchema
	c.RUnlock()

	// Check if we have a valid cached token that hasn't expired
	if existedToken != nil {
		// TODO: how to reduce the unnecessary re-login?
		// timeRemaining := time.Until(existedToken.Time)
		// if timeRemaining > expireTolerant {
		// 	return nil
		// }
	}

	_, err := c.maintainLogin(ctx, cli, up, param, math.MaxUint)
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
		return nil, fmt.Errorf("update prover token fail: %V", err)
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
