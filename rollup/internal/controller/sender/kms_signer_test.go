package sender

import (
	"context"
	"crypto/ecdsa"
	"encoding/asn1"
	"math/big"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/holiman/uint256"
	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scroll-tech/rollup/internal/config"
)

var (
	ecPublicKeyOID = asn1.ObjectIdentifier{1, 2, 840, 10045, 2, 1}
	secp256k1OID   = asn1.ObjectIdentifier{1, 3, 132, 0, 10}
)

// fakeKMS implements kmsAPI backed by a local secp256k1 key, mimicking the DER
// encodings a real AWS KMS ECC_SECG_P256K1 key returns.
type fakeKMS struct {
	priv    *ecdsa.PrivateKey
	forceHi bool // emit a high-s signature to exercise EIP-2 normalization
	signErr error
}

func (f *fakeKMS) GetPublicKey(_ context.Context, _ *kms.GetPublicKeyInput, _ ...func(*kms.Options)) (*kms.GetPublicKeyOutput, error) {
	pubBytes := crypto.FromECDSAPub(&f.priv.PublicKey) // 0x04 || X || Y
	der, err := asn1.Marshal(asn1Spki{
		Algorithm: asn1AlgorithmIdentifier{Algorithm: ecPublicKeyOID, Parameters: secp256k1OID},
		PublicKey: asn1.BitString{Bytes: pubBytes, BitLength: len(pubBytes) * 8},
	})
	if err != nil {
		return nil, err
	}
	return &kms.GetPublicKeyOutput{PublicKey: der}, nil
}

func (f *fakeKMS) Sign(_ context.Context, in *kms.SignInput, _ ...func(*kms.Options)) (*kms.SignOutput, error) {
	if f.signErr != nil {
		return nil, f.signErr
	}
	sig, err := crypto.Sign(in.Message, f.priv) // canonical low-s [R||S||V]
	if err != nil {
		return nil, err
	}
	r := new(big.Int).SetBytes(sig[0:32])
	s := new(big.Int).SetBytes(sig[32:64])
	if f.forceHi {
		s = new(big.Int).Sub(crypto.S256().Params().N, s) // make it high-s
	}
	der, err := asn1.Marshal(asn1EcSig{R: r, S: s})
	if err != nil {
		return nil, err
	}
	return &kms.SignOutput{Signature: der}, nil
}

func newTestKMSSigner(t *testing.T, fake *fakeKMS, chainID *big.Int) *kmsSigner {
	t.Helper()
	expected := crypto.PubkeyToAddress(fake.priv.PublicKey)
	ks, err := newKMSSignerWithClient(context.Background(), fake, "test-key-id", expected, chainID)
	require.NoError(t, err)
	assert.Equal(t, expected, ks.address())
	return ks
}

func TestKMSSigner_AddressValidation(t *testing.T) {
	priv, err := crypto.GenerateKey()
	require.NoError(t, err)
	fake := &fakeKMS{priv: priv}

	// matching address succeeds
	newTestKMSSigner(t, fake, big.NewInt(534352))

	// mismatching address fails fast
	_, err = newKMSSignerWithClient(context.Background(), fake, "test-key-id", common.HexToAddress("0xdeadbeef00000000000000000000000000000000"), big.NewInt(534352))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not match")
}

func TestKMSSigner_SignAllTxTypes(t *testing.T) {
	priv, err := crypto.GenerateKey()
	require.NoError(t, err)

	chainID := big.NewInt(534352)
	to := common.HexToAddress("0x000000000000000000000000000000000000dEaD")

	cases := []struct {
		name    string
		forceHi bool
		txData  gethTypes.TxData
	}{
		{
			name:   "legacy",
			txData: &gethTypes.LegacyTx{Nonce: 1, GasPrice: big.NewInt(1e9), Gas: 21000, To: &to, Value: big.NewInt(1)},
		},
		{
			name:   "dynamic_fee",
			txData: &gethTypes.DynamicFeeTx{ChainID: chainID, Nonce: 2, GasTipCap: big.NewInt(1e9), GasFeeCap: big.NewInt(2e9), Gas: 21000, To: &to, Value: big.NewInt(1)},
		},
		{
			name: "blob",
			txData: &gethTypes.BlobTx{
				ChainID:    uint256.MustFromBig(chainID),
				Nonce:      3,
				GasTipCap:  uint256.NewInt(1e9),
				GasFeeCap:  uint256.NewInt(2e9),
				Gas:        21000,
				To:         to,
				BlobFeeCap: uint256.NewInt(1e9),
				BlobHashes: []common.Hash{{0x01}},
				// A sidecar must survive signing — it carries the blobs/commitments
				// the node needs to accept the tx. Production blob txs always set it.
				Sidecar: &gethTypes.BlobTxSidecar{},
			},
		},
		{
			name:    "dynamic_fee_high_s",
			forceHi: true,
			txData:  &gethTypes.DynamicFeeTx{ChainID: chainID, Nonce: 4, GasTipCap: big.NewInt(1e9), GasFeeCap: big.NewInt(2e9), Gas: 21000, To: &to, Value: big.NewInt(1)},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ks := newTestKMSSigner(t, &fakeKMS{priv: priv, forceHi: tc.forceHi}, chainID)
			ts := &TransactionSigner{
				config:    &config.SignerConfig{SignerType: AWSKMSSignerType},
				kmsSigner: ks,
				addr:      ks.address(),
			}

			tx := gethTypes.NewTx(tc.txData)
			signedTx, err := ts.SignTransaction(context.Background(), tx)
			require.NoError(t, err)

			// recovered sender must match the KMS key address
			signer := gethTypes.LatestSignerForChainID(chainID)
			from, err := gethTypes.Sender(signer, signedTx)
			require.NoError(t, err)
			assert.Equal(t, ks.address(), from)

			// output signature must be canonical low-s regardless of what KMS returned.
			// RawSignatureValues returns (v, r, s) — the third value is s.
			_, _, s := signedTx.RawSignatureValues()
			assert.True(t, s.Cmp(secp256k1HalfN) <= 0, "signature s must be low-s")

			// a blob tx's sidecar must survive signing, otherwise the node rejects it.
			if tx.BlobTxSidecar() != nil {
				assert.NotNil(t, signedTx.BlobTxSidecar(), "blob sidecar must survive signing")
			}
		})
	}
}

// rawKMS returns operator-supplied raw bytes, used to exercise malformed
// GetPublicKey / Sign responses that the address-deriving fakeKMS can't produce.
type rawKMS struct {
	pub []byte
	sig []byte
}

func (f *rawKMS) GetPublicKey(_ context.Context, _ *kms.GetPublicKeyInput, _ ...func(*kms.Options)) (*kms.GetPublicKeyOutput, error) {
	return &kms.GetPublicKeyOutput{PublicKey: f.pub}, nil
}

func (f *rawKMS) Sign(_ context.Context, _ *kms.SignInput, _ ...func(*kms.Options)) (*kms.SignOutput, error) {
	return &kms.SignOutput{Signature: f.sig}, nil
}

func marshalSPKI(t *testing.T, algo, curve asn1.ObjectIdentifier, point []byte, bitLen int) []byte {
	t.Helper()
	der, err := asn1.Marshal(asn1Spki{
		Algorithm: asn1AlgorithmIdentifier{Algorithm: algo, Parameters: curve},
		PublicKey: asn1.BitString{Bytes: point, BitLength: bitLen},
	})
	require.NoError(t, err)
	return der
}

func TestKMSSigner_MalformedPublicKey(t *testing.T) {
	priv, err := crypto.GenerateKey()
	require.NoError(t, err)
	point := crypto.FromECDSAPub(&priv.PublicKey) // 65-byte uncompressed
	addr := crypto.PubkeyToAddress(priv.PublicKey)
	full := len(point) * 8

	// For the unused-bits case we need valid DER padding (the discarded low bits
	// must be zero) so that asn1.Unmarshal accepts it and our explicit BitLength
	// check is the thing that rejects it, deterministically.
	evenPoint := append([]byte(nil), point...)
	evenPoint[len(evenPoint)-1] &^= 1

	// control: a well-formed SPKI must still be accepted.
	_, err = newKMSSignerWithClient(context.Background(), &rawKMS{pub: marshalSPKI(t, ecPublicKeyOID, secp256k1OID, point, full)}, "k", addr, big.NewInt(534352))
	require.NoError(t, err)

	cases := []struct {
		name string
		pub  []byte
		want string
	}{
		{"wrong_algorithm_oid", marshalSPKI(t, secp256k1OID, secp256k1OID, point, full), "algorithm/curve"},
		{"wrong_curve_oid", marshalSPKI(t, ecPublicKeyOID, ecPublicKeyOID, point, full), "algorithm/curve"},
		{"trailing_bytes", append(marshalSPKI(t, ecPublicKeyOID, secp256k1OID, point, full), 0x00), "trailing bytes"},
		{"unused_bits", marshalSPKI(t, ecPublicKeyOID, secp256k1OID, evenPoint, full-1), "unused bits"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := newKMSSignerWithClient(context.Background(), &rawKMS{pub: tc.pub}, "k", addr, big.NewInt(534352))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestKMSSigner_MalformedSignature(t *testing.T) {
	addr := common.HexToAddress("0x1C5A77d9FA7eF466951B2F01F724BCa3A5820b63")
	hash := crypto.Keccak256([]byte("tx"))

	derOf := func(r, s *big.Int) []byte {
		b, err := asn1.Marshal(asn1EcSig{R: r, S: s})
		require.NoError(t, err)
		return b
	}

	cases := []struct {
		name string
		sig  []byte
		want string
	}{
		{"not_der", []byte{0x05, 0x00}, "failed to parse DER signature"},
		{"trailing_bytes", append(derOf(big.NewInt(1), big.NewInt(1)), 0x00), "trailing bytes"},
		{"zero_r", derOf(big.NewInt(0), big.NewInt(1)), "out of range"},
		{"negative_s", derOf(big.NewInt(1), big.NewInt(-1)), "out of range"},
		{"r_equals_n", derOf(new(big.Int).Set(secp256k1N), big.NewInt(1)), "out of range"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ks := &kmsSigner{client: &rawKMS{sig: tc.sig}, keyID: "k", addr: addr}
			_, err := ks.sign(context.Background(), hash)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestKMSSigner_InvalidSignerAddress(t *testing.T) {
	_, err := newKMSSigner(context.Background(), &config.AWSKMSSignerConfig{
		KeyID:         "some-key-id",
		SignerAddress: "not-a-hex-address",
	}, big.NewInt(534352))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a valid hex address")
}
