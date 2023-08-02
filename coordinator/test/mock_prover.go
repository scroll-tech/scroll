package test

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/logic/verifier"
	"scroll-tech/coordinator/internal/types"
)

type proofStatus uint32

const (
	verifiedSuccess proofStatus = iota
	verifiedFailed
	generatedFailed
)

type mockProver struct {
	proverName     string
	privKey        *ecdsa.PrivateKey
	proofType      message.ProofType
	coordinatorURL string
}

func newMockProver(t *testing.T, proverName string, coordinatorURL string, proofType message.ProofType) *mockProver {
	privKey, err := crypto.GenerateKey()
	assert.NoError(t, err)

	prover := &mockProver{
		proverName:     proverName,
		privKey:        privKey,
		proofType:      proofType,
		coordinatorURL: coordinatorURL,
	}
	return prover
}

// connectToCoordinator sets up a websocket client to connect to the prover manager.
func (r *mockProver) connectToCoordinator(t *testing.T) string {
	var loginResult types.LoginSchema
	client := resty.New()
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody([]byte(`{"prover_name":"mock_test"}`)).
		SetResult(&loginResult).
		Post(r.coordinatorURL + "/coordinator/v1/login")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	return loginResult.Token
}

func (r *mockProver) healthCheck(t *testing.T, token string) bool {
	var result types.Response
	client := resty.New()
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", token)).
		SetResult(&result).
		Get(r.coordinatorURL + "/coordinator/v1/health_check")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Equal(t, types.Success, result.ErrCode)
	return true
}

func (r *mockProver) getProverTask(t *testing.T, proofType message.ProofType) *types.ProverTaskSchema {
	// get task from coordinator
	token := r.connectToCoordinator(t)
	assert.NotEmpty(t, token)

	var result types.Response
	client := resty.New()
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", token)).
		SetBody(map[string]interface{}{"prover_version": 1, "prover_height": 100, "proof_type": int(proofType)}).
		SetResult(&result).
		Post(r.coordinatorURL + "/coordinator/v1/prover_tasks")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Equal(t, types.Success, result.ErrCode)

	data, ok := result.Data.(string)
	assert.True(t, ok)

	var proverTaskSchema types.ProverTaskSchema
	assert.NoError(t, json.Unmarshal([]byte(data), &proverTaskSchema))
	assert.NotEmpty(t, proverTaskSchema.TaskID)
	assert.NotEmpty(t, proverTaskSchema.ProofType)
	assert.NotEmpty(t, proverTaskSchema.ProofData)

	return &proverTaskSchema
}

func (r *mockProver) submitProof(t *testing.T, proverTaskSchema *types.ProverTaskSchema, proofStatus proofStatus) {
	proof := &message.ProofMsg{
		ProofDetail: &message.ProofDetail{
			ID:         proverTaskSchema.TaskID,
			Type:       message.ProofType(proverTaskSchema.ProofType),
			Status:     message.StatusOk,
			ChunkProof: &message.ChunkProof{},
			BatchProof: &message.BatchProof{},
		},
	}

	if proofStatus == generatedFailed {
		proof.Status = message.StatusProofError
	} else if proofStatus == verifiedFailed {
		proof.ProofDetail.ChunkProof.Proof = []byte(verifier.InvalidTestProof)
		proof.ProofDetail.BatchProof.Proof = []byte(verifier.InvalidTestProof)
	}

	assert.NoError(t, proof.Sign(r.privKey))
	submitProof := types.SubmitProofParameter{
		TaskID:    proof.ID,
		ProofType: int(proof.Type),
		Status:    int(proof.Status),
		Signature: proof.Signature,
	}

	switch proof.Type {
	case message.ProofTypeChunk:
		encodeData, err := json.Marshal(proof.ChunkProof)
		assert.NoError(t, err)
		assert.NotEmpty(t, encodeData)
		submitProof.Proof = string(encodeData)
	case message.ProofTypeBatch:
		encodeData, err := json.Marshal(proof.BatchProof)
		assert.NoError(t, err)
		assert.NotEmpty(t, encodeData)
		submitProof.Proof = string(encodeData)
	}

	token := r.connectToCoordinator(t)
	assert.NotEmpty(t, token)

	submitProofData, err := json.Marshal(submitProof)
	assert.NoError(t, err)
	assert.NotNil(t, submitProofData)

	var result types.Response
	client := resty.New()
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", token)).
		SetBody(string(submitProofData)).
		SetResult(&result).
		Post(r.coordinatorURL + "/coordinator/v1/prover_tasks")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Equal(t, types.Success, result.ErrCode)
}
