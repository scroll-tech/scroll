package test

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/mitchellh/mapstructure"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"

	ctypes "scroll-tech/common/types"
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
	proverVersion  string
	privKey        *ecdsa.PrivateKey
	proofType      message.ProofType
	coordinatorURL string
}

func newMockProver(t *testing.T, proverName string, coordinatorURL string, proofType message.ProofType, version string) *mockProver {
	privKey, err := crypto.GenerateKey()
	assert.NoError(t, err)

	prover := &mockProver{
		proverName:     proverName,
		proverVersion:  version,
		privKey:        privKey,
		proofType:      proofType,
		coordinatorURL: coordinatorURL,
	}
	return prover
}

// connectToCoordinator sets up a websocket client to connect to the prover manager.
func (r *mockProver) connectToCoordinator(t *testing.T, proverTypes []types.ProverType) (string, int, string) {
	challengeString := r.challenge(t)
	return r.login(t, challengeString, proverTypes)
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

func (r *mockProver) login(t *testing.T, challengeString string, proverTypes []types.ProverType) (string, int, string) {
	authMsg := types.LoginParameter{
		Message: types.Message{
			Challenge:     challengeString,
			ProverName:    r.proverName,
			ProverVersion: r.proverVersion,
			ProverTypes:   proverTypes,
			VKs:           []string{"mock_vk"},
		},
		PublicKey: r.publicKey(),
	}
	assert.NoError(t, authMsg.SignWithKey(r.privKey))
	body, err := json.Marshal(authMsg)
	assert.NoError(t, err)

	var result ctypes.Response
	client := resty.New()
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", challengeString)).
		SetBody(body).
		SetResult(&result).
		Post("http://" + r.coordinatorURL + "/coordinator/v1/login")
	assert.NoError(t, err)

	if result.ErrCode != 0 {
		return "", result.ErrCode, result.ErrMsg
	}

	type login struct {
		Time  string `json:"time"`
		Token string `json:"token"`
	}
	var loginData login
	err = mapstructure.Decode(result.Data, &loginData)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Empty(t, result.ErrMsg)
	return loginData.Token, 0, ""
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

func (r *mockProver) getProverTask(t *testing.T, proofType message.ProofType) (*types.GetTaskSchema, int, string) {
	// get task from coordinator
	token, errCode, errMsg := r.connectToCoordinator(t, []types.ProverType{types.MakeProverType(proofType)})
	if errCode != 0 {
		return nil, errCode, errMsg
	}
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
		SetBody(map[string]interface{}{"prover_height": 100, "task_types": []int{int(proofType)}}).
		SetResult(&result).
		Post("http://" + r.coordinatorURL + "/coordinator/v1/get_task")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	return &result.Data, result.ErrCode, result.ErrMsg
}

// Testing expected errors returned by coordinator.
//
//nolint:unparam
func (r *mockProver) tryGetProverTask(t *testing.T, proofType message.ProofType) (int, string) {
	// get task from coordinator
	token, errCode, errMsg := r.connectToCoordinator(t, []types.ProverType{types.MakeProverType(proofType)})
	if errCode != 0 {
		return errCode, errMsg
	}
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

	var proof []byte
	switch proverTaskSchema.TaskType {
	case int(message.ProofTypeChunk):
		encodeData, err := json.Marshal(message.ChunkProof{})
		assert.NoError(t, err)
		assert.NotEmpty(t, encodeData)
		proof = encodeData
	case int(message.ProofTypeBatch):
		encodeData, err := json.Marshal(message.BatchProof{})
		assert.NoError(t, err)
		assert.NotEmpty(t, encodeData)
		proof = encodeData
	}

	if proofStatus == verifiedFailed {
		switch proverTaskSchema.TaskType {
		case int(message.ProofTypeChunk):
			chunkProof := message.ChunkProof{}
			chunkProof.Proof = []byte(verifier.InvalidTestProof)
			encodeData, err := json.Marshal(&chunkProof)
			assert.NoError(t, err)
			assert.NotEmpty(t, encodeData)
			proof = encodeData
		case int(message.ProofTypeBatch):
			batchProof := message.BatchProof{}
			batchProof.Proof = []byte(verifier.InvalidTestProof)
			encodeData, err := json.Marshal(&batchProof)
			assert.NoError(t, err)
			assert.NotEmpty(t, encodeData)
			proof = encodeData
		}
	}

	submitProof := types.SubmitProofParameter{
		UUID:     proverTaskSchema.UUID,
		TaskID:   proverTaskSchema.TaskID,
		TaskType: proverTaskSchema.TaskType,
		Status:   int(proofMsgStatus),
		Proof:    string(proof),
	}

	token, authErrCode, errMsg := r.connectToCoordinator(t, []types.ProverType{types.MakeProverType(message.ProofType(proverTaskSchema.TaskType))})
	assert.Equal(t, authErrCode, 0)
	assert.Equal(t, errMsg, "")
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
