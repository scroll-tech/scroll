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

	coordinatorType "scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
)

// NewVerifier Sets up a rust ffi to call verify.
func NewVerifier(cfg *config.VerifierConfig) (*Verifier, error) {
	if cfg.MockMode {
		batchVKMap := map[string]string{cfg.ForkName: "mock_vk"}
		chunkVKMap := map[string]string{cfg.ForkName: "mock_vk"}
		return &Verifier{cfg: cfg, ChunkVKMap: chunkVKMap, BatchVKMap: batchVKMap}, nil
	}
	paramsPathStr := C.CString(cfg.ParamsPath)
	assetsPathStr := C.CString(cfg.AssetsPath)
	defer func() {
		C.free(unsafe.Pointer(paramsPathStr))
		C.free(unsafe.Pointer(assetsPathStr))
	}()

	C.init_batch_verifier(paramsPathStr, assetsPathStr)
	C.init_chunk_verifier(paramsPathStr, assetsPathStr)

	v := &Verifier{
		cfg:        cfg,
		ChunkVKMap: make(map[string]string),
		BatchVKMap: make(map[string]string),
	}

	bundleVK, err := v.readVK(path.Join(cfg.AssetsPath, "bundle_vk.vkey"))
	if err != nil {
		return nil, err
	}
	batchVK, err := v.readVK(path.Join(cfg.AssetsPath, "batch_vk.vkey"))
	if err != nil {
		return nil, err
	}
	chunkVK, err := v.readVK(path.Join(cfg.AssetsPath, "chunk_vk.vkey"))
	if err != nil {
		return nil, err
	}
	v.BundleVkMap[cfg.ForkName] = bundleVK
	v.BatchVKMap[cfg.ForkName] = batchVK
	v.ChunkVKMap[cfg.ForkName] = chunkVK

	if err := v.loadEmbedVK(); err != nil {
		return nil, err
	}
	return v, nil
}

// VerifyBatchProof Verify a ZkProof by marshaling it and sending it to the Halo2 Verifier.
func (v *Verifier) VerifyBatchProof(proof *coordinatorType.BatchProof, forkName string) (bool, error) {
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
	defer func() {
		C.free(unsafe.Pointer(proofStr))
		C.free(unsafe.Pointer(forkNameStr))
	}()

	verified := C.verify_batch_proof(proofStr, forkNameStr)
	return verified != 0, nil
}

// VerifyChunkProof Verify a ZkProof by marshaling it and sending it to the Halo2 Verifier.
func (v *Verifier) VerifyChunkProof(proof *coordinatorType.ChunkProof) (bool, error) {
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
	defer func() {
		C.free(unsafe.Pointer(proofStr))
	}()

	log.Info("Start to verify chunk proof ...")
	verified := C.verify_chunk_proof(proofStr)
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
	batchVKBytes, err := fs.ReadFile(legacyVKFS, "legacy_vk/agg_vk.vkey")
	if err != nil {
		log.Error("load embed batch vk failure", "err", err)
		return err
	}

	chunkVkBytes, err := fs.ReadFile(legacyVKFS, "legacy_vk/chunk_vk.vkey")
	if err != nil {
		log.Error("load embed chunk vk failure", "err", err)
		return err
	}

	v.BatchVKMap["curie"] = base64.StdEncoding.EncodeToString(batchVKBytes)
	v.ChunkVKMap["curie"] = base64.StdEncoding.EncodeToString(chunkVkBytes)
	return nil
}
