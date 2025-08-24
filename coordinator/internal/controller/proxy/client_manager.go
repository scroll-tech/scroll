package proxy

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/types"
)

type Client interface {
	Client(context.Context) *upClient
}

type ClientManager struct {
	cliCfg  *config.ProxyClient
	cfg     *config.UpStream
	privKey *ecdsa.PrivateKey
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

func (cliMgr *ClientManager) Client(ctx context.Context) *upClient {
	return newUpClient(cliMgr.cfg)
}

func (cliMgr *ClientManager) generateLoginParameter(privKey []byte, challenge string) (*types.LoginParameter, error) {

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
