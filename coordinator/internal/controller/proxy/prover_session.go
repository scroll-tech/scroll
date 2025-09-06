package proxy

import (
	"context"
	"fmt"
	"sync"

	ctypes "scroll-tech/common/types"
	"scroll-tech/coordinator/internal/types"
)

// Client wraps an http client with a preset host for coordinator API calls
type proverSession struct {
	sync.RWMutex
	proverToken   string
	phase         uint
	completionCtx context.Context
}

func (c *proverSession) maintainLogin(ctx context.Context, cliMgr Client, param *types.LoginParameter, phase uint) error {
	c.Lock()
	curPhase := c.phase
	if c.completionCtx != nil {
		waitctx := c.completionCtx
		c.Unlock()
		select {
		case <-waitctx.Done():
			return c.maintainLogin(ctx, cliMgr, param, phase)
		case <-ctx.Done():
			return fmt.Errorf("ctx fail")
		}
	}

	if phase < curPhase {
		// outdate login phase, give up
		c.Unlock()
		return nil
	}

	// occupy the update slot
	completeCtx, cf := context.WithCancel(ctx)
	defer cf()
	c.completionCtx = completeCtx
	c.Unlock()

	cli := cliMgr.Client(ctx)
	if cli == nil {
		return fmt.Errorf("get upstream cli fail")
	}

	resp, err := cli.ProxyLogin(ctx, param)
	if err != nil {
		return fmt.Errorf("proxylogin fail: %v", err)
	}

	if resp.ErrCode == ctypes.ErrJWTTokenExpired {
		cliMgr.Reset(cli)
		cli = cliMgr.Client(ctx)
		if cli == nil {
			return fmt.Errorf("get upstream cli fail (secondary try)")
		}

		// like SDK, we would try one more time if the upstream token is expired
		resp, err = cli.ProxyLogin(ctx, param)
		if err != nil {
			return fmt.Errorf("proxylogin fail: %v", err)
		}
	}

	if resp.ErrCode != 0 {
		return fmt.Errorf("upstream fail: %d (%s)", resp.ErrCode, resp.ErrMsg)
	}

	var loginResult types.LoginSchema
	if err := resp.DecodeData(&loginResult); err != nil {
		return err
	}

	c.Lock()
	defer c.Unlock()
	c.proverToken = loginResult.Token
	c.completionCtx = nil

	return nil
}

// ProxyLogin makes a POST request to /v1/proxy_login with LoginParameter
func (c *proverSession) ProxyLogin(ctx context.Context, cli Client, param *types.LoginParameter) error {
	c.RLock()
	phase := c.phase + 1
	c.RUnlock()

	return c.maintainLogin(ctx, cli, param, phase)
}

// GetTask makes a POST request to /v1/get_task with GetTaskParameter
func (c *proverSession) GetTask(ctx context.Context, param *types.GetTaskParameter, cliMgr Client) (*ctypes.Response, error) {
	c.RLock()
	phase := c.phase
	token := c.proverToken
	c.RUnlock()

	cli := cliMgr.Client(ctx)
	if cli == nil {
		return nil, fmt.Errorf("get upstream cli fail")
	}

	resp, err := cli.GetTask(ctx, param, token)
	if err != nil {
		return nil, err
	}

	if resp.ErrCode == ctypes.ErrJWTTokenExpired {
		// get param from ctx
		loginParam, ok := ctx.Value(LoginParamCache).(*types.LoginParameter)
		if !ok {
			return nil, fmt.Errorf("Unexpected error, no loginparam ctx value")
		}

		err = c.maintainLogin(ctx, cliMgr, loginParam, phase)
		if err != nil {
			return nil, fmt.Errorf("update prover token fail: %V", err)
		}

		// like SDK, we would try one more time if the upstream token is expired
		return cli.GetTask(ctx, param, token)
	}

	return resp, nil
}

// SubmitProof makes a POST request to /v1/submit_proof with SubmitProofParameter
func (c *proverSession) SubmitProof(ctx context.Context, param *types.SubmitProofParameter, cliMgr Client) (*ctypes.Response, error) {
	c.RLock()
	phase := c.phase
	token := c.proverToken
	c.RUnlock()

	cli := cliMgr.Client(ctx)
	if cli == nil {
		return nil, fmt.Errorf("get upstream cli fail")
	}

	resp, err := cli.SubmitProof(ctx, param, token)
	if err != nil {
		return nil, err
	}

	if resp.ErrCode == ctypes.ErrJWTTokenExpired {
		// get param from ctx
		loginParam, ok := ctx.Value(LoginParamCache).(*types.LoginParameter)
		if !ok {
			return nil, fmt.Errorf("Unexpected error, no loginparam ctx value")
		}

		err = c.maintainLogin(ctx, cliMgr, loginParam, phase)
		if err != nil {
			return nil, fmt.Errorf("update prover token fail: %V", err)
		}

		// like SDK, we would try one more time if the upstream token is expired
		return cli.SubmitProof(ctx, param, token)
	}

	return resp, nil
}
