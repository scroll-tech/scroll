package sender

import (
	"context"
	"crypto/ecdsa"
	"encoding/asn1"
	"fmt"
	"math/big"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kmstypes "github.com/aws/aws-sdk-go-v2/service/kms/types"

	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/rollup/internal/config"
)

// kmsAPI captures the subset of the AWS KMS client used by the signer. It is an
// interface so the signer can be unit-tested with a mock in place of a live KMS.
type kmsAPI interface {
	GetPublicKey(ctx context.Context, params *kms.GetPublicKeyInput, optFns ...func(*kms.Options)) (*kms.GetPublicKeyOutput, error)
	Sign(ctx context.Context, params *kms.SignInput, optFns ...func(*kms.Options)) (*kms.SignOutput, error)
}

// kmsSigner signs transactions with an AWS KMS asymmetric secp256k1 key. The
// private key never leaves KMS: only the 32-byte signing hash is sent, and KMS
// returns a DER-encoded signature which we turn into Ethereum's 65-byte
// [R || S || V] form and apply to the transaction locally. Because the tx is
// assembled locally, every tx type the sender builds is supported, including BlobTx.
type kmsSigner struct {
	client   kmsAPI
	keyID    string
	addr     common.Address
	txSigner gethTypes.Signer
}

// asn1AlgorithmIdentifier is the AlgorithmIdentifier of a SubjectPublicKeyInfo.
type asn1AlgorithmIdentifier struct {
	Algorithm  asn1.ObjectIdentifier
	Parameters asn1.ObjectIdentifier
}

// asn1Spki mirrors the SubjectPublicKeyInfo DER structure returned by KMS
// GetPublicKey for an ECC_SECG_P256K1 key. PublicKey holds the uncompressed
// point (0x04 || X || Y).
type asn1Spki struct {
	Algorithm asn1AlgorithmIdentifier
	PublicKey asn1.BitString
}

// asn1EcSig mirrors the DER ECDSA signature KMS returns: SEQUENCE { r, s }.
type asn1EcSig struct {
	R *big.Int
	S *big.Int
}

// OIDs expected in the SPKI AlgorithmIdentifier of a KMS ECC_SECG_P256K1 key.
var (
	oidPublicKeyECDSA = asn1.ObjectIdentifier{1, 2, 840, 10045, 2, 1} // id-ecPublicKey
	oidNamedCurveS256 = asn1.ObjectIdentifier{1, 3, 132, 0, 10}       // secp256k1
)

// secp256k1N is the secp256k1 curve order; secp256k1HalfN is N/2. Signatures with
// s above N/2 are normalized to N-s (EIP-2 low-s).
var (
	secp256k1N     = crypto.S256().Params().N
	secp256k1HalfN = new(big.Int).Rsh(secp256k1N, 1)
)

// newKMSSigner constructs a signer backed by a live AWS KMS key. cfg.SignerAddress
// is required and validated against the address derived from the KMS public key,
// so a wrong key id fails fast instead of signing from an unexpected account.
func newKMSSigner(ctx context.Context, cfg *config.AWSKMSSignerConfig, chainID *big.Int) (*kmsSigner, error) {
	if cfg == nil {
		return nil, fmt.Errorf("aws_kms_signer_config is nil")
	}
	if cfg.KeyID == "" {
		return nil, fmt.Errorf("aws kms signer: key_id is empty")
	}
	if cfg.SignerAddress == "" {
		return nil, fmt.Errorf("aws kms signer: signer_address is required")
	}
	if !common.IsHexAddress(cfg.SignerAddress) {
		return nil, fmt.Errorf("aws kms signer: signer_address %q is not a valid hex address", cfg.SignerAddress)
	}

	var optFns []func(*awsconfig.LoadOptions) error
	if cfg.Region != "" {
		optFns = append(optFns, awsconfig.WithRegion(cfg.Region))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, optFns...)
	if err != nil {
		return nil, fmt.Errorf("aws kms signer: failed to load aws config: %w", err)
	}
	if awsCfg.Region == "" {
		return nil, fmt.Errorf("aws kms signer: aws region is not set (configure region or AWS_REGION)")
	}

	return newKMSSignerWithClient(ctx, kms.NewFromConfig(awsCfg), cfg.KeyID, common.HexToAddress(cfg.SignerAddress), chainID)
}

// newKMSSignerWithClient is the testable core: it derives the key's address from
// the KMS public key and asserts it matches expectedAddr.
func newKMSSignerWithClient(ctx context.Context, client kmsAPI, keyID string, expectedAddr common.Address, chainID *big.Int) (*kmsSigner, error) {
	pub, err := publicKeyFromKMS(ctx, client, keyID)
	if err != nil {
		return nil, err
	}
	derivedAddr := crypto.PubkeyToAddress(*pub)
	if derivedAddr != expectedAddr {
		return nil, fmt.Errorf("aws kms signer: configured signer_address %s does not match address %s derived from KMS key %s", expectedAddr.Hex(), derivedAddr.Hex(), keyID)
	}
	log.Info("initialized AWS KMS signer", "keyID", keyID, "address", derivedAddr.Hex(), "chainID", chainID)
	return &kmsSigner{
		client:   client,
		keyID:    keyID,
		addr:     derivedAddr,
		txSigner: gethTypes.LatestSignerForChainID(chainID),
	}, nil
}

func publicKeyFromKMS(ctx context.Context, client kmsAPI, keyID string) (*ecdsa.PublicKey, error) {
	out, err := client.GetPublicKey(ctx, &kms.GetPublicKeyInput{KeyId: &keyID})
	if err != nil {
		return nil, fmt.Errorf("aws kms signer: GetPublicKey failed: %w", err)
	}
	var spki asn1Spki
	rest, err := asn1.Unmarshal(out.PublicKey, &spki)
	if err != nil {
		return nil, fmt.Errorf("aws kms signer: failed to parse public key DER: %w", err)
	}
	if len(rest) != 0 {
		return nil, fmt.Errorf("aws kms signer: trailing bytes after public key DER")
	}
	if !spki.Algorithm.Algorithm.Equal(oidPublicKeyECDSA) || !spki.Algorithm.Parameters.Equal(oidNamedCurveS256) {
		return nil, fmt.Errorf("aws kms signer: unexpected public key algorithm/curve, want id-ecPublicKey/secp256k1")
	}
	if spki.PublicKey.BitLength != len(spki.PublicKey.Bytes)*8 {
		return nil, fmt.Errorf("aws kms signer: public key bit string has unused bits")
	}
	// crypto.UnmarshalPubkey requires the 65-byte uncompressed form (0x04 || X || Y)
	// and verifies the point lies on the curve.
	pub, err := crypto.UnmarshalPubkey(spki.PublicKey.Bytes)
	if err != nil {
		return nil, fmt.Errorf("aws kms signer: failed to unmarshal secp256k1 public key: %w", err)
	}
	return pub, nil
}

// address returns the Ethereum address of the KMS key.
func (k *kmsSigner) address() common.Address {
	return k.addr
}

// signTx returns tx signed by the KMS key.
func (k *kmsSigner) signTx(ctx context.Context, tx *gethTypes.Transaction) (*gethTypes.Transaction, error) {
	sig, err := k.sign(ctx, k.txSigner.Hash(tx).Bytes())
	if err != nil {
		return nil, err
	}
	signedTx, err := tx.WithSignature(k.txSigner, sig)
	if err != nil {
		return nil, fmt.Errorf("aws kms signer: failed to apply signature to tx: %w", err)
	}
	return signedTx, nil
}

// sign returns the 65-byte [R || S || V] Ethereum signature over the given 32-byte hash.
func (k *kmsSigner) sign(ctx context.Context, hash []byte) ([]byte, error) {
	out, err := k.client.Sign(ctx, &kms.SignInput{
		KeyId:            &k.keyID,
		Message:          hash,
		MessageType:      kmstypes.MessageTypeDigest,
		SigningAlgorithm: kmstypes.SigningAlgorithmSpecEcdsaSha256,
	})
	if err != nil {
		return nil, fmt.Errorf("aws kms signer: Sign failed: %w", err)
	}

	var sig asn1EcSig
	rest, err := asn1.Unmarshal(out.Signature, &sig)
	if err != nil {
		return nil, fmt.Errorf("aws kms signer: failed to parse DER signature: %w", err)
	}
	if len(rest) != 0 {
		return nil, fmt.Errorf("aws kms signer: trailing bytes after DER signature")
	}
	if sig.R == nil || sig.S == nil {
		return nil, fmt.Errorf("aws kms signer: DER signature missing r or s")
	}

	r, s := sig.R, sig.S
	// Guard against a malformed DER signature: r and s must be in [1, N-1].
	// Besides being the valid ECDSA range, this keeps them positive and <32 bytes,
	// so FillBytes below cannot panic.
	if r.Sign() <= 0 || s.Sign() <= 0 || r.Cmp(secp256k1N) >= 0 || s.Cmp(secp256k1N) >= 0 {
		return nil, fmt.Errorf("aws kms signer: signature (r,s) out of range [1, N-1]")
	}
	// EIP-2: enforce low-s to keep signatures canonical.
	if s.Cmp(secp256k1HalfN) > 0 {
		s = new(big.Int).Sub(secp256k1N, s)
	}

	rsSig := make([]byte, 65)
	r.FillBytes(rsSig[0:32])
	s.FillBytes(rsSig[32:64])

	// Recover the recovery id (v). KMS returns only (r, s), so we try the two valid
	// Ethereum parities and keep the one that recovers our address. Recovery ids 2/3
	// require R.x >= N (probability ~2^-128) and are not valid Ethereum yParity values,
	// so an unrecoverable signature returns the error below rather than a bad signature.
	for v := byte(0); v <= 1; v++ {
		rsSig[64] = v
		pub, err := crypto.SigToPub(hash, rsSig)
		if err != nil {
			continue
		}
		if crypto.PubkeyToAddress(*pub) == k.addr {
			return rsSig, nil
		}
	}
	return nil, fmt.Errorf("aws kms signer: failed to recover a valid recovery id for the signature")
}
