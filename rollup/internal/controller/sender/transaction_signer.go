package sender

import (
	"context"
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"

	"scroll-tech/rollup/internal/config"
)

const (
	// PrivateKeySignerType is the type of signer that uses a private key to sign transactions
	PrivateKeySignerType = "PrivateKey"

	// RemoteSignerType is the type of signer that uses a remote signer to sign transactions
	RemoteSignerType = "RemoteSigner"
)

// TransactionSigner signs given transactions
type TransactionSigner struct {
	config    *config.SignerConfig
	auth      *bind.TransactOpts
	rpcClient *rpc.Client
	nonce     uint64
	addr      common.Address
}

func NewTransactionSigner(config *config.SignerConfig, chainID *big.Int) (*TransactionSigner, error) {
	switch config.SignerType {
	case PrivateKeySignerType:
		privKey, err := crypto.ToECDSA(common.FromHex(config.PrivateKeySignerConfig.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("parse sender private key failed: %w", err)
		}
		auth, err := bind.NewKeyedTransactorWithChainID(privKey, chainID)
		if err != nil {
			return nil, fmt.Errorf("failed to create transactor with chain ID %v, err: %w", chainID, err)
		}
		return &TransactionSigner{
			config: config,
			auth:   auth,
			addr:   crypto.PubkeyToAddress(privKey.PublicKey),
		}, nil
	case RemoteSignerType:
		if config.RemoteSignerConfig.SignerAddress == "" {
			return nil, fmt.Errorf("failed to create RemoteSigner, signer address is empty")
		}
		rpcClient, err := rpc.Dial(config.RemoteSignerConfig.RemoteSignerUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to dial rpc client, err: %w", err)
		}
		return &TransactionSigner{
			config:    config,
			rpcClient: rpcClient,
			addr:      common.HexToAddress(config.RemoteSignerConfig.SignerAddress),
		}, nil
	default:
		return nil, fmt.Errorf("failed to create new transaction signer, unknown type: %v", config.SignerType)
	}
}

func (ts *TransactionSigner) SignTransaction(ctx context.Context, tx *gethTypes.Transaction) (*gethTypes.Transaction, error) {
	switch ts.config.SignerType {
	case PrivateKeySignerType:
		signedTx, err := ts.auth.Signer(ts.addr, tx)
		if err != nil {
			log.Info("failed to sign tx", "address", ts.addr.String(), "err", err)
			return nil, err
		}
		return signedTx, nil
	case RemoteSignerType:
		rpcTx, err := txDataToRpcTx(&ts.addr, tx)
		if err != nil {
			return nil, fmt.Errorf("failed to convert txData to rpc transaction, err: %w", err)
		}
		var result hexutil.Bytes
		err = ts.rpcClient.CallContext(ctx, &result, "eth_signTransaction", rpcTx)
		if err != nil {
			log.Info("failed to call remote rpc", "err", err)
			return nil, err
		}
		signedTx := new(gethTypes.Transaction)
		if err := signedTx.UnmarshalBinary(result); err != nil {
			return nil, err
		}
		return signedTx, nil
	default:
		// this shouldn't happen, because SignerType is checked during creation
		return nil, fmt.Errorf("shouldn't happen, unknown signer type")
	}
}

func (ts *TransactionSigner) SetNonce(nonce uint64) {
	ts.nonce = nonce
}

func (ts *TransactionSigner) GetNonce() uint64 {
	return ts.nonce
}

func (ts *TransactionSigner) GetAddr() common.Address {
	return ts.addr
}

func (ts *TransactionSigner) GetType() string {
	return ts.config.SignerType
}

// RpcTransaction transaction that will be send through rpc to web3Signer
type RpcTransaction struct {
	From                 *common.Address `json:"from"`
	To                   *common.Address `json:"to"`
	Gas                  uint64          `json:"gas"`
	GasPrice             *big.Int        `json:"gasPrice,omitempty"`
	MaxPriorityFeePerGas *big.Int        `json:"maxPriorityFeePerGas,omitempty"`
	MaxFeePerGas         *big.Int        `json:"maxFeePerGas,omitempty"`
	Nonce                uint64          `json:"nonce"`
	Value                *big.Int        `json:"value"`
	Data                 string          `json:"data"`
}

func txDataToRpcTx(from *common.Address, tx *gethTypes.Transaction) (*RpcTransaction, error) {
	switch tx.Type() {
	case gethTypes.LegacyTxType:
		return &RpcTransaction{
			From:     from,
			To:       tx.To(),
			Gas:      tx.Gas(),
			GasPrice: tx.GasPrice(),
			Nonce:    tx.Nonce(),
			Value:    tx.Value(),
			Data:     common.Bytes2Hex(tx.Data()),
		}, nil
	case gethTypes.DynamicFeeTxType:
		return &RpcTransaction{
			From:                 from,
			To:                   tx.To(),
			Gas:                  tx.Gas(),
			MaxPriorityFeePerGas: tx.GasTipCap(),
			MaxFeePerGas:         tx.GasFeeCap(),
			Nonce:                tx.Nonce(),
			Value:                tx.Value(),
			Data:                 common.Bytes2Hex(tx.Data()),
		}, nil
	default: // other tx types (BlobTx) currently not supported by web3signer
		return nil, fmt.Errorf("failed to convert tx to RpcTransaction, unsupported tx type, %d", tx.Type())
	}
}
