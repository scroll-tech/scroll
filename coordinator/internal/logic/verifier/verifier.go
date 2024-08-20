//go:build !mock_verifier

package verifier

/*
#cgo LDFLAGS: -lzkp -lm -ldl -L${SRCDIR}/lib/ -Wl,-rpath=${SRCDIR}/lib
#cgo gpu LDFLAGS: -lzkp -lm -ldl -lgmp -lstdc++ -lprocps -L/usr/local/cuda/lib64/ -lcudart -L${SRCDIR}/lib/ -Wl,-rpath=${SRCDIR}/lib
#include <stdlib.h>
#include "./lib/libzkp.h"
*/
import "C" //nolint:typecheck

import (
	"embed"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path"
	"unsafe"

	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
)

type rustVerifierConfig struct {
	LowVersionCircuit  *config.CircuitConfig `json:"low_version_circuit"`
	HighVersionCircuit *config.CircuitConfig `json:"high_version_circuit"`
}

// NewVerifier Sets up a rust ffi to call verify.
func NewVerifier(cfg *config.VerifierConfig) (*Verifier, error) {
	if cfg.MockMode {
		chunkVKMap := map[string]string{cfg.ForkName: "mock_vk"}
		batchVKMap := map[string]string{cfg.ForkName: "mock_vk"}
		bundleVKMap := map[string]string{cfg.ForkName: "mock_vk"}
		return &Verifier{cfg: cfg, ChunkVKMap: chunkVKMap, BatchVKMap: batchVKMap, BundleVkMap: bundleVKMap}, nil
	}
	verifierConfig := rustVerifierConfig{
		LowVersionCircuit:  cfg.LowVersionCircuit,
		HighVersionCircuit: cfg.HighVersionCircuit,
	}
	configBytes, err := json.Marshal(verifierConfig)
	if err != nil {
		return nil, err
	}

	configStr := C.CString(string(configBytes))
	assetsPathHiStr := C.CString(cfg.HighVersionCircuit.AssetsPath)
	defer func() {
		C.free(unsafe.Pointer(configStr))
		C.free(unsafe.Pointer(assetsPathHiStr))
	}()

	C.init(configStr)

	v := &Verifier{
		cfg:         cfg,
		ChunkVKMap:  make(map[string]struct{}),
		BatchVKMap:  make(map[string]struct{}),
		BundleVkMap: make(map[string]struct{}),
	}

	bundleVK, err := v.readVK(path.Join(cfg.HighVersionCircuit.AssetsPath, "vk_bundle.vkey"))
	if err != nil {
		return nil, err
	}
	batchVK, err := v.readVK(path.Join(cfg.HighVersionCircuit.AssetsPath, "vk_batch.vkey"))
	if err != nil {
		return nil, err
	}
	chunkVK, err := v.readVK(path.Join(cfg.HighVersionCircuit.AssetsPath, "vk_chunk.vkey"))
	if err != nil {
		return nil, err
	}
	v.BundleVkMap[bundleVK] = struct{}{}
	v.BatchVKMap[batchVK] = struct{}{}
	v.ChunkVKMap[chunkVK] = struct{}{}

	if err := v.loadEmbedVK(); err != nil {
		return nil, err
	}
	return v, nil
}

// VerifyBatchProof Verify a ZkProof by marshaling it and sending it to the Halo2 Verifier.
func (v *Verifier) VerifyBatchProof(proof *message.BatchProof, forkName, circuitsVersion string) (bool, error) {
	if v.cfg.MockMode {
		log.Info("Mock mode, batch verifier disabled")
		if string(proof.Proof) == InvalidTestProof {
			return false, nil
		}
		return true, nil

	}
	buf, err := json.Marshal(proof)
	if err != nil {
		return false, err
	}

	log.Info("Start to verify batch proof", "forkName", forkName)
	proofStr := C.CString(string(buf))
	forkNameStr := C.CString(forkName)
	circuitsVersionStr := C.CString(circuitsVersion)
	defer func() {
		C.free(unsafe.Pointer(proofStr))
		C.free(unsafe.Pointer(forkNameStr))
		C.free(unsafe.Pointer(circuitsVersionStr))
	}()

	verified := C.verify_batch_proof(proofStr, forkNameStr, circuitsVersionStr)
	return verified != 0, nil
}

// VerifyChunkProof Verify a ZkProof by marshaling it and sending it to the Halo2 Verifier.
func (v *Verifier) VerifyChunkProof(proof *message.ChunkProof, forkName, circuitsVersion string) (bool, error) {
	if v.cfg.MockMode {
		log.Info("Mock mode, verifier disabled")
		if string(proof.Proof) == InvalidTestProof {
			return false, nil
		}
		return true, nil

	}
	buf, err := json.Marshal(proof)
	if err != nil {
		return false, err
	}

	log.Info("Start to verify chunk proof", "forkName", forkName)
	proofStr := C.CString(string(buf))
	forkNameStr := C.CString(forkName)
	circuitsVersionStr := C.CString(circuitsVersion)
	defer func() {
		C.free(unsafe.Pointer(proofStr))
		C.free(unsafe.Pointer(forkNameStr))
		C.free(unsafe.Pointer(circuitsVersionStr))
	}()

	verified := C.verify_chunk_proof(proofStr, forkNameStr, circuitsVersionStr)
	return verified != 0, nil
}

// VerifyBundleProof Verify a ZkProof for a bundle of batches, by marshaling it and verifying it via the EVM verifier.
func (v *Verifier) VerifyBundleProof(proof *message.BundleProof, forkName, circuitsVersion string) (bool, error) {
	if v.cfg.MockMode {
		log.Info("Mock mode, verifier disabled")
		if string(proof.Proof) == InvalidTestProof {
			return false, nil
		}
		return true, nil

	}
	buf, err := json.Marshal(proof)
	if err != nil {
		return false, err
	}

	proofStr := C.CString(string(buf))
	forkNameStr := C.CString(forkName)
	circuitsVersionStr := C.CString(circuitsVersion)
	defer func() {
		C.free(unsafe.Pointer(proofStr))
		C.free(unsafe.Pointer(forkNameStr))
		C.free(unsafe.Pointer(circuitsVersionStr))
	}()

	log.Info("Start to verify bundle proof ...")
	verified := C.verify_bundle_proof(proofStr, forkNameStr, circuitsVersionStr)
	return verified != 0, nil
}

func (v *Verifier) readVK(filePat string) (string, error) {
	f, err := os.Open(filePat)
	if err != nil {
		return "", err
	}
	byt, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(byt), nil
}

//go:embed legacy_vk/*
var legacyVKFS embed.FS

func (v *Verifier) loadEmbedVK() error {
	chunkVKBytes, err := fs.ReadFile(legacyVKFS, "legacy_vk/vk_chunk.vkey")
	if err != nil {
		log.Error("load embed chunk vk failure", "err", err)
		return err
	}

	batchVKBytes, err := fs.ReadFile(legacyVKFS, "legacy_vk/vk_batch.vkey")
	if err != nil {
		log.Error("load embed batch vk failure", "err", err)
		return err
	}

	bundleVKBytes, err := fs.ReadFile(legacyVKFS, "legacy_vk/vk_bundle.vkey")
	if err != nil {
		log.Error("load embed chunk vk failure", "err", err)
		return err
	}

	v.ChunkVKMap[base64.StdEncoding.EncodeToString(chunkVKBytes)] = struct{}{}
	v.BatchVKMap[base64.StdEncoding.EncodeToString(batchVKBytes)] = struct{}{}
	v.BundleVkMap[base64.StdEncoding.EncodeToString(bundleVKBytes)] = struct{}{}

	return nil
}
