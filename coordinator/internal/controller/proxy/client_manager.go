package proxy

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"net/http"
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
	PeekClient() *upClient
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

func (cliMgr *ClientManager) doLogin(ctx context.Context, loginCli *upClient) time.Time {
	// Calculate wait time between 2 seconds and cfg.RetryWaitTime
	minWait := 2 * time.Second
	waitDuration := time.Duration(cliMgr.cfg.RetryWaitTime) * time.Second
	if waitDuration < minWait {
		waitDuration = minWait
	}

	for {
		log.Info("attempting login to upstream coordinator", "name", cliMgr.name)
		loginResult, err := loginCli.Login(ctx)
		if err == nil && loginResult != nil {
			log.Info("login to upstream coordinator successful", "name", cliMgr.name, "time", loginResult.Time)
			return loginResult.Time
		}
		log.Info("login to upstream coordinator failed, retrying", "name", cliMgr.name, "error", err, "waitDuration", waitDuration)

		timer := time.NewTimer(waitDuration)
		select {
		case <-ctx.Done():
			timer.Stop()
			return time.Now()
		case <-timer.C:
			// Continue to next retry
		}
	}
}

func (cliMgr *ClientManager) PeekClient() *upClient {
	cliMgr.cachedCli.RLock()
	defer cliMgr.cachedCli.RUnlock()

	return cliMgr.cachedCli.cli
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
		loginCli := newUpClient(cliMgr.cfg, cliMgr)
		cliMgr.cachedCli.completionCtx = context.WithValue(ctx, "cli", loginCli)

		// Launch login goroutine
		go func() {
			defer completionDone()
			expiredT := cliMgr.doLogin(context.Background(), loginCli)

			cliMgr.cachedCli.Lock()
			cliMgr.cachedCli.cli = loginCli
			cliMgr.cachedCli.completionCtx = nil

			// Launch waiting thread to clear cached client before expiration
			go func() {
				now := time.Now()
				clearTime := expiredT.Add(-10 * time.Second) // 10s before expiration

				// If clear time is too soon (less than 10s from now), set it to 10s from now
				if clearTime.Before(now.Add(10 * time.Second)) {
					clearTime = now.Add(10 * time.Second)
					log.Error("token expiration time is too close, delaying clear time",
						"name", cliMgr.name,
						"expiredT", expiredT,
						"adjustedClearTime", clearTime)
				}

				waitDuration := time.Until(clearTime)
				log.Info("token expiration monitor started",
					"name", cliMgr.name,
					"expiredT", expiredT,
					"clearTime", clearTime,
					"waitDuration", waitDuration)

				timer := time.NewTimer(waitDuration)
				select {
				case <-ctx.Done():
					timer.Stop()
					log.Info("token expiration monitor cancelled", "name", cliMgr.name)
				case <-timer.C:
					log.Info("clearing cached client before token expiration",
						"name", cliMgr.name,
						"expiredT", expiredT)
					cliMgr.clearCachedCli(loginCli)
				}
			}()

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

func (cliMgr *ClientManager) clearCachedCli(cli *upClient) {
	cliMgr.cachedCli.Lock()
	if cliMgr.cachedCli.cli == cli {
		cliMgr.cachedCli.cli = nil
		cliMgr.cachedCli.completionCtx = nil
		log.Info("cached client cleared due to forbidden response", "name", cliMgr.name)
	}
	cliMgr.cachedCli.Unlock()
}

func (cliMgr *ClientManager) OnResp(cli *upClient, resp *http.Response) {
	if resp.StatusCode == http.StatusForbidden {
		log.Info("cached client cleared due to forbidden response", "name", cliMgr.name)
		cliMgr.clearCachedCli(cli)
	}
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
