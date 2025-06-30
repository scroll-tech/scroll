package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"scroll-tech/coordinator/internal/logic/verifier"

	"scroll-tech/common/types/message"

	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

func verify(cCtx *cli.Context) error {
	var forkName, proofType, proofPath string
	if cCtx.Args().Len() <= 2 {
		forkName = cfg.ProverManager.Verifier.Verifiers[0].ForkName
		proofType = cCtx.Args().First()
		proofPath = cCtx.Args().Get(1)
	} else {
		forkName = cCtx.Args().First()
		proofType = cCtx.Args().Get(1)
		proofPath = cCtx.Args().Get(2)
	}
	log.Info("verify proof in: ", proofPath, "type", proofType, "forkName", forkName)

	// Load the content of the proof file
	data, err := os.ReadFile(filepath.Clean(proofPath))
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	vf, err := verifier.NewVerifier(cfg.ProverManager.Verifier)
	if err != nil {
		return err
	}

	var ret bool
	switch strings.ToLower(proofType) {
	case "chunk":
		proof := &message.OpenVMChunkProof{}
		if err := json.Unmarshal(data, proof); err != nil {
			return err
		}
		vk, ok := vf.ChunkVk[forkName]
		if !ok {
			return fmt.Errorf("no vk loaded for fork %s", forkName)
		}
		if len(proof.Vk) != 0 {
			if bytes.Equal(proof.Vk, vk) {
				return fmt.Errorf("unmatch vk with expected: expected %s, get %s",
					base64.StdEncoding.EncodeToString(vk),
					base64.StdEncoding.EncodeToString(proof.Vk),
				)
			}
		} else {
			proof.Vk = vk
		}

		ret, err = vf.VerifyChunkProof(proof, forkName)
	case "batch":
		proof := &message.OpenVMBatchProof{}
		if err := json.Unmarshal(data, proof); err != nil {
			return err
		}
		vk, ok := vf.BatchVk[forkName]
		if !ok {
			return fmt.Errorf("no vk loaded for fork %s", forkName)
		}
		if len(proof.Vk) != 0 {
			if bytes.Equal(proof.Vk, vk) {
				return fmt.Errorf("unmatch vk with expected: expected %s, get %s",
					base64.StdEncoding.EncodeToString(vk),
					base64.StdEncoding.EncodeToString(proof.Vk),
				)
			}
		} else {
			proof.Vk = vk
		}

		ret, err = vf.VerifyBatchProof(proof, forkName)
	case "bundle":
		proof := &message.OpenVMBundleProof{}
		if err := json.Unmarshal(data, proof); err != nil {
			return err
		}
		vk, ok := vf.BundleVk[forkName]
		if !ok {
			return fmt.Errorf("no vk loaded for fork %s", forkName)
		}
		if len(proof.Vk) != 0 {
			if bytes.Equal(proof.Vk, vk) {
				return fmt.Errorf("unmatch vk with expected: expected %s, get %s",
					base64.StdEncoding.EncodeToString(vk),
					base64.StdEncoding.EncodeToString(proof.Vk),
				)
			}
		} else {
			proof.Vk = vk
		}

		ret, err = vf.VerifyBundleProof(proof, forkName)
	default:
		return fmt.Errorf("unsupport proof type %s", proofType)
	}

	if err != nil {
		return err
	}
	log.Info("verified:", "ret", ret)
	return nil
}
