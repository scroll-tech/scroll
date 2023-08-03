package test

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/mitchellh/mapstructure"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"

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
	randomString := r.random(t)
	return r.login(t, randomString)
}

func (r *mockProver) random(t *testing.T) string {
	var result types.Response
	client := resty.New()
	resp, err := client.R().
		SetResult(&result).
		Get("http://" + r.coordinatorURL + "/coordinator/v1/random")
	assert.NoError(t, err)

	type login struct {
		Time  string `json:"time"`
		Token string `json:"token"`
	}
	var loginData login
	err = mapstructure.Decode(result.Data, &loginData)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Empty(t, result.ErrMsg)
	return loginData.Token
}

func (r *mockProver) login(t *testing.T, randomString string) string {
	authMsg := message.AuthMsg{
		Identity: &message.Identity{
			Random:     randomString,
			ProverName: "test",
		},
	}
	assert.NoError(t, authMsg.SignWithKey(r.privKey))

	body := fmt.Sprintf("{\"message\":{\"random\":\"%s\",\"prover_name\":\"%s\"},\"signature\":\"%s\"}",
		authMsg.Identity.Random, authMsg.Identity.ProverName, authMsg.Signature)

	var result types.Response
	client := resty.New()
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", randomString)).
		SetBody([]byte(body)).
		SetResult(&result).
		Post("http://" + r.coordinatorURL + "/coordinator/v1/login")
	assert.NoError(t, err)

	type login struct {
		Time  string `json:"time"`
		Token string `json:"token"`
	}
	var loginData login
	err = mapstructure.Decode(result.Data, &loginData)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Empty(t, result.ErrMsg)
	return loginData.Token
}

func (r *mockProver) healthCheck(t *testing.T, token string, errCode int) bool {
	var result types.Response
	client := resty.New()
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", token)).
		SetResult(&result).
		Get("http://" + r.coordinatorURL + "/coordinator/v1/health_check")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Equal(t, errCode, result.ErrCode)
	return true
}

func (r *mockProver) getProverTask(t *testing.T, proofType message.ProofType) *types.ProverTaskSchema {
	// get task from coordinator
	token := r.connectToCoordinator(t)
	assert.NotEmpty(t, token)

	type response struct {
		ErrCode int                    `json:"errcode"`
		ErrMsg  string                 `json:"errmsg"`
		Data    types.ProverTaskSchema `json:"data"`
	}

	var result response
	client := resty.New()
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", token)).
		SetBody(map[string]interface{}{"prover_version": 1, "prover_height": 100, "proof_type": int(proofType)}).
		SetResult(&result).
		Post("http://" + r.coordinatorURL + "/coordinator/v1/prover_tasks")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Equal(t, types.Success, result.ErrCode)

	assert.NotEmpty(t, result.Data.TaskID)
	assert.NotEmpty(t, result.Data.ProofType)
	assert.NotEmpty(t, result.Data.ProofData)
	return &result.Data
}

func (r *mockProver) submitProof(t *testing.T, proverTaskSchema *types.ProverTaskSchema, proofStatus proofStatus) {
	proof := &message.ProofMsg{
		ProofDetail: &message.ProofDetail{
			ID:         proverTaskSchema.TaskID,
			Type:       message.ProofType(proverTaskSchema.ProofType),
			Status:     message.RespStatus(proofStatus),
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
		Post("http://" + r.coordinatorURL + "/coordinator/v1/submit_proof")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Equal(t, types.Success, result.ErrCode)
}
