package proxy

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"sync"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/types"
)

type Client interface {
	Client(context.Context) *upClient
	Reset(cli *upClient)
	Name() string
}

type ClientManager struct {
	name    string
	cliCfg  *config.ProxyClient
	cfg     *config.UpStream
	privKey *ecdsa.PrivateKey

	cachedCli struct {
		sync.RWMutex
		cli           *upClient
		completionCtx context.Context
	}
}

// transformToValidPrivateKey safely transforms arbitrary bytes into valid private key bytes
func buildPrivateKey(inputBytes []byte) (*ecdsa.PrivateKey, error) {
	// Try appending bytes from 0x0 to 0x20 until we get a valid private key
	for appendByte := byte(0x0); appendByte <= 0x20; appendByte++ {
		// Append the byte to input
		extendedBytes := append(inputBytes, appendByte)

		// Calculate 256-bit hash
		hash := crypto.Keccak256(extendedBytes)

		// Try to create private key from hash
		if k, err := crypto.ToECDSA(hash); err == nil {
			return k, nil
		}
	}

	return nil, fmt.Errorf("failed to generate valid private key from input bytes")
}

func NewClientManager(name string, cliCfg *config.ProxyClient, cfg *config.UpStream) (*ClientManager, error) {

	log.Info("init client", "name", name, "upcfg", cfg.BaseUrl, "compatible mode", cfg.CompatibileMode)
	privKey, err := buildPrivateKey([]byte(cliCfg.Secret))
	if err != nil {
		return nil, err
	}

	return &ClientManager{
		name:    name,
		privKey: privKey,
		cfg:     cfg,
		cliCfg:  cliCfg,
	}, nil
}

func (cliMgr *ClientManager) doLogin(ctx context.Context, loginCli *upClient) {
	if cliMgr.cfg.CompatibileMode {
		loginCli.loginToken = "dummy"
		log.Info("Skip login process for compatibile mode")
		return
	}

	// Calculate wait time between 2 seconds and cfg.RetryWaitTime
	minWait := 2 * time.Second
	waitDuration := time.Duration(cliMgr.cfg.RetryWaitTime) * time.Second
	if waitDuration < minWait {
		waitDuration = minWait
	}

	for {
		log.Info("attempting login to upstream coordinator", "name", cliMgr.name)
		loginResp, err := loginCli.Login(ctx, cliMgr.genLoginParam)
		if err == nil && loginResp.ErrCode == 0 {
			var loginResult loginSchema
			err = loginResp.DecodeData(&loginResult)
			if err != nil {
				log.Error("login parsing data fail", "error", err)
			} else {
				loginCli.loginToken = loginResult.Token
				log.Info("login to upstream coordinator successful", "name", cliMgr.name, "time", loginResult.Time)
				// TODO: we need to parse time if we start making use of it
				return
			}
		} else if err != nil {
			log.Error("login process fail", "error", err)
		} else {
			log.Error("login get fail resp", "code", loginResp.ErrCode, "msg", loginResp.ErrMsg)
		}

		log.Info("login to upstream coordinator failed, retrying", "name", cliMgr.name, "error", err, "waitDuration", waitDuration)

		timer := time.NewTimer(waitDuration)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			// Continue to next retry
		}
	}
}

func (cliMgr *ClientManager) Reset(cli *upClient) {
	cliMgr.cachedCli.Lock()
	if cliMgr.cachedCli.cli == cli {
		cliMgr.cachedCli.cli = nil
	}
	cliMgr.cachedCli.Unlock()
	log.Info("cached client cleared", "name", cliMgr.name)
}

func (cliMgr *ClientManager) Name() string {
	return cliMgr.name
}

func (cliMgr *ClientManager) Client(ctx context.Context) *upClient {
	cliMgr.cachedCli.RLock()
	if cliMgr.cachedCli.cli != nil {
		defer cliMgr.cachedCli.RUnlock()
		return cliMgr.cachedCli.cli
	}
	cliMgr.cachedCli.RUnlock()

	cliMgr.cachedCli.Lock()
	if cliMgr.cachedCli.cli != nil {
		defer cliMgr.cachedCli.Unlock()
		return cliMgr.cachedCli.cli
	}

	var completionCtx context.Context
	// Check if completion context is set
	if cliMgr.cachedCli.completionCtx != nil {
		completionCtx = cliMgr.cachedCli.completionCtx
	} else {
		// Set new completion context and launch login goroutine
		ctx, completionDone := context.WithCancel(context.TODO())
		loginCli := newUpClient(cliMgr.cfg)
		completionCtx = context.WithValue(ctx, "cli", loginCli)
		cliMgr.cachedCli.completionCtx = completionCtx

		// Launch keep-login goroutine
		go func() {
			defer completionDone()
			cliMgr.doLogin(context.Background(), loginCli)

			cliMgr.cachedCli.Lock()
			cliMgr.cachedCli.cli = loginCli
			cliMgr.cachedCli.completionCtx = nil

			cliMgr.cachedCli.Unlock()

		}()
	}
	cliMgr.cachedCli.Unlock()

	// Wait for completion or request cancellation
	select {
	case <-ctx.Done():
		return nil
	case <-completionCtx.Done():
		cli := completionCtx.Value("cli").(*upClient)
		return cli
	}
}

func (cliMgr *ClientManager) genLoginParam(challenge string) (*types.LoginParameter, error) {

	// Generate public key string
	publicKeyHex := common.Bytes2Hex(crypto.CompressPubkey(&cliMgr.privKey.PublicKey))

	// Create login parameter with proxy settings
	loginParam := &types.LoginParameter{
		Message: types.Message{
			Challenge:          challenge,
			ProverName:         cliMgr.cliCfg.ProxyName,
			ProverVersion:      cliMgr.cliCfg.ProxyVersion,
			ProverProviderType: types.ProverProviderTypeProxy,
			ProverTypes:        []types.ProverType{}, // Default empty
			VKs:                []string{},           // Default empty
		},
		PublicKey: publicKeyHex,
	}

	// Sign the message with the private key
	if err := loginParam.SignWithKey(cliMgr.privKey); err != nil {
		return nil, fmt.Errorf("failed to sign login parameter: %w", err)
	}

	return loginParam, nil
}
