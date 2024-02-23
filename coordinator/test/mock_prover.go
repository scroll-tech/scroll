package test

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-resty/resty/v2"
	"github.com/mitchellh/mapstructure"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"

	ctypes "scroll-tech/common/types"
	"scroll-tech/common/types/message"
	"scroll-tech/common/version"

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
	proverVersion  string
	privKey        *ecdsa.PrivateKey
	proofType      message.ProofType
	coordinatorURL string
}

func newMockProver(t *testing.T, proverName string, coordinatorURL string, proofType message.ProofType) *mockProver {
	privKey, err := crypto.GenerateKey()
	assert.NoError(t, err)

	prover := &mockProver{
		proverName:     proverName,
		proverVersion:  version.Version,
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
	var result ctypes.Response
	client := resty.New()
	resp, err := client.R().
		SetResult(&result).
		Get("http://" + r.coordinatorURL + "/coordinator/v1/challenge")
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
			ProverName:    r.proverName,
			ProverVersion: r.proverVersion,
		},
	}
	assert.NoError(t, authMsg.SignWithKey(r.privKey))

	body := fmt.Sprintf("{\"message\":{\"challenge\":\"%s\",\"prover_name\":\"%s\", \"prover_version\":\"%s\"},\"signature\":\"%s\"}",
		authMsg.Identity.Challenge, authMsg.Identity.ProverName, authMsg.Identity.ProverVersion, authMsg.Signature)

	var result ctypes.Response
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

func (r *mockProver) healthCheckSuccess(t *testing.T) bool {
	var result ctypes.Response
	client := resty.New()
	resp, err := client.R().
		SetResult(&result).
		Get("http://" + r.coordinatorURL + "/coordinator/v1/challenge")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Equal(t, ctypes.Success, result.ErrCode)
	return true
}

func (r *mockProver) healthCheckFailure(t *testing.T) bool {
	var result ctypes.Response
	client := resty.New()
	resp, err := client.R().
		SetResult(&result).
		Get("http://" + r.coordinatorURL + "/coordinator/v1/challenge")
	assert.Error(t, err)
	assert.Equal(t, 0, resp.StatusCode())
	assert.Equal(t, 0, result.ErrCode)
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

// Testing expected errors returned by coordinator.
func (r *mockProver) tryGetProverTask(t *testing.T, proofType message.ProofType) (int, string) {
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

	return result.ErrCode, result.ErrMsg
}

func (r *mockProver) submitProof(t *testing.T, proverTaskSchema *types.GetTaskSchema, proofStatus proofStatus, errCode int) {
	proofMsgStatus := message.StatusOk
	if proofStatus == generatedFailed {
		proofMsgStatus = message.StatusProofError
	}

	proof := &message.ProofMsg{
		ProofDetail: &message.ProofDetail{
			ID:         proverTaskSchema.TaskID,
			Type:       message.ProofType(proverTaskSchema.TaskType),
			Status:     proofMsgStatus,
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

	var result ctypes.Response
	client := resty.New()
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", token)).
		SetBody(string(submitProofData)).
		SetResult(&result).
		Post("http://" + r.coordinatorURL + "/coordinator/v1/submit_proof")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Equal(t, errCode, result.ErrCode)
}

func (r *mockProver) publicKey() string {
	return common.Bytes2Hex(crypto.CompressPubkey(&r.privKey.PublicKey))
}
