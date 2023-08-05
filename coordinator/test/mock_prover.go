package test

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/mitchellh/mapstructure"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"

	ctypes "scroll-tech/common/types"
	"scroll-tech/common/utils"

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
	challengeString := r.challenge(t)
	return r.login(t, challengeString)
}

func (r *mockProver) challenge(t *testing.T) string {
	var result types.Response
	var resp *resty.Response
	var err error

	client := resty.New()
	utils.TryTimes(10, func() bool {
		resp, err = client.R().
			SetResult(&result).
			Get("http://" + r.coordinatorURL + "/coordinator/v1/challenge")
		if err != nil {
			return false
		}
		return true
	})
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

func (r *mockProver) login(t *testing.T, challengeString string) string {
	authMsg := message.AuthMsg{
		Identity: &message.Identity{
			Challenge:     challengeString,
			ProverName:    "test",
			ProverVersion: "v1.0.0",
		},
	}
	assert.NoError(t, authMsg.SignWithKey(r.privKey))

	body := fmt.Sprintf("{\"message\":{\"challenge\":\"%s\",\"prover_name\":\"%s\", \"prover_version\":\"%s\"},\"signature\":\"%s\"}",
		authMsg.Identity.Challenge, authMsg.Identity.ProverName, authMsg.Identity.ProverVersion, authMsg.Signature)

	var result types.Response
	client := resty.New()
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", challengeString)).
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
		Get("http://" + r.coordinatorURL + "/coordinator/v1/healthz")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Equal(t, errCode, result.ErrCode)
	return true
}

func (r *mockProver) getProverTask(t *testing.T, proofType message.ProofType) *types.GetTaskSchema {
	// get task from coordinator
	token := r.connectToCoordinator(t)
	assert.NotEmpty(t, token)

	type response struct {
		ErrCode int                 `json:"errcode"`
		ErrMsg  string              `json:"errmsg"`
		Data    types.GetTaskSchema `json:"data"`
	}

	var result response
	client := resty.New()
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", token)).
		SetBody(map[string]interface{}{"prover_height": 100, "task_type": int(proofType)}).
		SetResult(&result).
		Post("http://" + r.coordinatorURL + "/coordinator/v1/get_task")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Equal(t, ctypes.Success, result.ErrCode)

	assert.NotEmpty(t, result.Data.TaskID)
	assert.NotEmpty(t, result.Data.TaskType)
	assert.NotEmpty(t, result.Data.TaskData)
	return &result.Data
}

func (r *mockProver) submitProof(t *testing.T, proverTaskSchema *types.GetTaskSchema, proofStatus proofStatus) {
	proof := &message.ProofMsg{
		ProofDetail: &message.ProofDetail{
			ID:         proverTaskSchema.TaskID,
			Type:       message.ProofType(proverTaskSchema.TaskType),
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
		TaskID:   proof.ID,
		TaskType: int(proof.Type),
		Status:   int(proof.Status),
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
	assert.Equal(t, ctypes.Success, result.ErrCode)
}
