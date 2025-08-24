package proxy

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/types"
)

type Client interface {
	Client(*gin.Context) *upClient
}

type ClientManager struct {
	cliCfg  *config.ProxyClient
	cfg     *config.UpStream
	privKey *ecdsa.PrivateKey

	cachedCli struct {
		sync.RWMutex
		cli            *upClient
		completionCtx  context.Context
		completionDone context.CancelFunc
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

func NewClientManager(cliCfg *config.ProxyClient, cfg *config.UpStream) (*ClientManager, error) {

	privKey, err := buildPrivateKey([]byte(cliCfg.Auth.Secret))
	if err != nil {
		return nil, err
	}

	return &ClientManager{
		privKey: privKey,
		cfg:     cfg,
		cliCfg:  cliCfg,
	}, nil
}

func (cliMgr *ClientManager) doLogin() *upClient {
	loginCli := newUpClient(cliMgr.cfg, cliMgr)

	return loginCli
}

func (cliMgr *ClientManager) Client(ctx *gin.Context) *upClient {
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
		cliMgr.cachedCli.completionCtx = ctx

		// Launch login goroutine
		go func() {
			defer completionDone()

			loginCli := cliMgr.doLogin()
			if loginResult, err := loginCli.Login(context.Background()); err == nil {
				loginCli.loginToken = loginResult.Token

				cliMgr.cachedCli.Lock()
				cliMgr.cachedCli.cli = loginCli
				cliMgr.cachedCli.completionCtx = nil
				cliMgr.cachedCli.Unlock()
			}
		}()
	}
	cliMgr.cachedCli.Unlock()

	// Wait for completion or request cancellation
	select {
	case <-ctx.Done():
		return nil
	case <-completionCtx.Done():
		cliMgr.cachedCli.Lock()
		cli := cliMgr.cachedCli.cli
		cliMgr.cachedCli.Unlock()
		return cli
	}
}

func (cliMgr *ClientManager) OnError(isUnauth bool) {

}

func (cliMgr *ClientManager) GenLoginParam(challenge string) (*types.LoginParameter, error) {

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
