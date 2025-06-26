//go:build !mock_verifier

package verifier

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/libzkp"
)

// This struct maps to `CircuitConfig` in libzkp/impl/src/verifier.rs
// Define a brand new struct here is to eliminate side effects in case fields
// in `*config.CircuitConfig` being changed
type rustCircuitConfig struct {
	ForkName   string `json:"fork_name"`
	AssetsPath string `json:"assets_path"`
}

func newRustCircuitConfig(cfg *config.CircuitConfig) *rustCircuitConfig {
	return &rustCircuitConfig{
		ForkName:   cfg.ForkName,
		AssetsPath: cfg.AssetsPath,
	}
}

// This struct maps to `VerifierConfig` in coordinator/internal/logic/libzkp/impl/src/verifier.rs
// Define a brand new struct here is to eliminate side effects in case fields
// in `*config.VerifierConfig` being changed
type rustVerifierConfig struct {
	HighVersionCircuit *rustCircuitConfig `json:"high_version_circuit"`
}

func newRustVerifierConfig(cfg *config.VerifierConfig) *rustVerifierConfig {
	return &rustVerifierConfig{
		HighVersionCircuit: newRustCircuitConfig(cfg.HighVersionCircuit),
	}
}

type rustVkDump struct {
	Chunk  string `json:"chunk_vk"`
	Batch  string `json:"batch_vk"`
	Bundle string `json:"bundle_vk"`
}

// NewVerifier Sets up a rust ffi to call verify.
func NewVerifier(cfg *config.VerifierConfig) (*Verifier, error) {
	verifierConfig := newRustVerifierConfig(cfg)
	configBytes, err := json.Marshal(verifierConfig)
	if err != nil {
		return nil, err
	}

	libzkp.InitVerifier(string(configBytes))

	v := &Verifier{
		cfg:         cfg,
		OpenVMVkMap: make(map[string]struct{}),
	}

	if err := v.loadOpenVMVks(message.EuclidV2Fork); err != nil {
		return nil, err
	}

	return v, nil
}

// VerifyBatchProof Verify a ZkProof by marshaling it and sending it to the Verifier.
func (v *Verifier) VerifyBatchProof(proof *message.OpenVMBatchProof, forkName string) (bool, error) {
	buf, err := json.Marshal(proof)
	if err != nil {
		return false, err
	}

	log.Info("Start to verify batch proof", "forkName", forkName)
	return libzkp.VerifyBatchProof(string(buf), forkName), nil
}

// VerifyChunkProof Verify a ZkProof by marshaling it and sending it to the Verifier.
func (v *Verifier) VerifyChunkProof(proof *message.OpenVMChunkProof, forkName string) (bool, error) {
	buf, err := json.Marshal(proof)
	if err != nil {
		return false, err
	}

	log.Info("Start to verify chunk proof", "forkName", forkName)

	return libzkp.VerifyChunkProof(string(buf), forkName), nil
}

// VerifyBundleProof Verify a ZkProof for a bundle of batches, by marshaling it and verifying it via the EVM verifier.
func (v *Verifier) VerifyBundleProof(proof *message.OpenVMBundleProof, forkName string) (bool, error) {
	buf, err := json.Marshal(proof)
	if err != nil {
		return false, err
	}

	log.Info("Start to verify bundle proof ...")
	return libzkp.VerifyBundleProof(string(buf), forkName), nil
}

func (v *Verifier) ReadVK(filePat string) (string, error) {

	f, err := os.Open(filepath.Clean(filePat))
	if err != nil {
		return "", err
	}
	byt, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(byt), nil
}

func (v *Verifier) loadOpenVMVks(forkName string) error {
	tempFile := path.Join(os.TempDir(), "openVmVk.json")
	err := libzkp.DumpVk(forkName, tempFile)
	if err != nil {
		return err
	}

	f, err := os.Open(filepath.Clean(tempFile))
	if err != nil {
		return err
	}
	byt, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	var dump rustVkDump
	if err := json.Unmarshal(byt, &dump); err != nil {
		return err
	}
	v.OpenVMVkMap[dump.Chunk] = struct{}{}
	v.OpenVMVkMap[dump.Batch] = struct{}{}
	v.OpenVMVkMap[dump.Bundle] = struct{}{}
	return nil
}
