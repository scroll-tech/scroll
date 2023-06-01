package api

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/proof"
	"scroll-tech/coordinator/internal/logic/rollermanager"
	"scroll-tech/coordinator/internal/logic/verifier"
	"scroll-tech/coordinator/internal/orm"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

func geneAuthMsg(t *testing.T) *message.AuthMsg {
	authMsg := &message.AuthMsg{
		Identity: &message.Identity{
			Name:      "roller_test1",
			Timestamp: uint32(time.Now().Unix()),
		},
	}
	privKey, err := crypto.GenerateKey()
	assert.NoError(t, err)
	assert.NoError(t, authMsg.SignWithKey(privKey))
	return authMsg
}

var rollerController *RollerController

func init() {
	conf := &config.Config{}
	conf.RollerManagerConfig = &config.RollerManagerConfig{
		TokenTimeToLive: 120,
	}
	conf.RollerManagerConfig.Verifier = &config.VerifierConfig{MockMode: true}
	rollerController = NewRollerController(conf, nil)
}

func TestManager_RequestToken(t *testing.T) {
	convey.Convey("auth msg verify failure", t, func() {
		tmpAuthMsg := &message.AuthMsg{
			Identity: &message.Identity{
				Name:      "roller_test_request_token",
				Timestamp: uint32(time.Now().Unix()),
			},
		}
		token, err := rollerController.RequestToken(tmpAuthMsg)
		assert.Error(t, err)
		assert.Empty(t, token)
	})

	convey.Convey("token has already been distributed", t, func() {
		tmpAuthMsg := geneAuthMsg(t)
		key, _ := tmpAuthMsg.PublicKey()
		tokenCacheStored := "c393987bb791dd285dd3d8ffbd770ed1"
		rollerController.tokenCache.Set(key, tokenCacheStored, time.Hour)
		token, err := rollerController.RequestToken(tmpAuthMsg)
		assert.NoError(t, err)
		assert.Equal(t, token, tokenCacheStored)
	})

	convey.Convey("token generation failure", t, func() {
		tmpAuthMsg := geneAuthMsg(t)
		patchGuard := gomonkey.ApplyFunc(message.GenerateToken, func() (string, error) {
			return "", errors.New("token generation failed")
		})
		defer patchGuard.Reset()
		token, err := rollerController.RequestToken(tmpAuthMsg)
		assert.Error(t, err)
		assert.Empty(t, token)
	})

	convey.Convey("token generation success", t, func() {
		tmpAuthMsg := geneAuthMsg(t)
		tokenCacheStored := "c393987bb791dd285dd3d8ffbd770ed1"
		patchGuard := gomonkey.ApplyFunc(message.GenerateToken, func() (string, error) {
			return tokenCacheStored, nil
		})
		defer patchGuard.Reset()
		token, err := rollerController.RequestToken(tmpAuthMsg)
		assert.NoError(t, err)
		assert.Equal(t, tokenCacheStored, token)
	})
}

func TestManager_Register(t *testing.T) {
	convey.Convey("auth msg verify failure", t, func() {
		tmpAuthMsg := &message.AuthMsg{
			Identity: &message.Identity{
				Name:      "roller_test_register",
				Timestamp: uint32(time.Now().Unix()),
			},
		}
		subscription, err := rollerController.Register(context.Background(), tmpAuthMsg)
		assert.Error(t, err)
		assert.Empty(t, subscription)
	})

	convey.Convey("verify token failure", t, func() {
		tmpAuthMsg := geneAuthMsg(t)
		patchGuard := gomonkey.ApplyMethodFunc(rollerController, "VerifyToken", func(tmpAuthMsg *message.AuthMsg) (bool, error) {
			return false, errors.New("verify token failure")
		})
		defer patchGuard.Reset()
		subscription, err := rollerController.Register(context.Background(), tmpAuthMsg)
		assert.Error(t, err)
		assert.Empty(t, subscription)
	})

	convey.Convey("notifier failure", t, func() {
		tmpAuthMsg := geneAuthMsg(t)
		patchGuard := gomonkey.ApplyMethodFunc(rollerController, "VerifyToken", func(tmpAuthMsg *message.AuthMsg) (bool, error) {
			return true, nil
		})
		defer patchGuard.Reset()
		patchGuard.ApplyFunc(rpc.NotifierFromContext, func(ctx context.Context) (*rpc.Notifier, bool) {
			return nil, false
		})
		subscription, err := rollerController.Register(context.Background(), tmpAuthMsg)
		assert.Error(t, err)
		assert.Equal(t, err, rpc.ErrNotificationsUnsupported)
		assert.Equal(t, *subscription, rpc.Subscription{})
	})

	convey.Convey("register failure", t, func() {
		tmpAuthMsg := geneAuthMsg(t)
		patchGuard := gomonkey.ApplyMethodFunc(rollerController, "VerifyToken", func(tmpAuthMsg *message.AuthMsg) (bool, error) {
			return true, nil
		})
		defer patchGuard.Reset()

		var taskWorker *proof.TaskWorker
		patchGuard.ApplyPrivateMethod(taskWorker, "AllocTaskWorker", func(ctx context.Context, authMsg *message.AuthMsg) (*rpc.Subscription, error) {
			return nil, errors.New("register error")
		})
		subscription, err := rollerController.Register(context.Background(), tmpAuthMsg)
		assert.Error(t, err)
		assert.Empty(t, subscription)
	})

	convey.Convey("register success", t, func() {
		tmpAuthMsg := geneAuthMsg(t)
		patchGuard := gomonkey.ApplyMethodFunc(rollerController, "VerifyToken", func(tmpAuthMsg *message.AuthMsg) (bool, error) {
			return true, nil
		})
		defer patchGuard.Reset()

		var taskWorker *proof.TaskWorker
		patchGuard.ApplyPrivateMethod(taskWorker, "AllocTaskWorker", func(ctx context.Context, authMsg *message.AuthMsg) (*rpc.Subscription, error) {
			return nil, nil
		})
		_, err := rollerController.Register(context.Background(), tmpAuthMsg)
		assert.Error(t, err)
	})
}

func TestManager_SubmitProof(t *testing.T) {
	id := "10000"
	tmpProof := &message.ProofMsg{
		ProofDetail: &message.ProofDetail{
			Type:   message.BasicProve,
			ID:     id,
			Status: message.StatusOk,
			Proof:  &message.AggProof{},
		},
	}
	tmpAuthMsg := &message.AuthMsg{
		Identity: &message.Identity{
			Name:      "roller_test_register",
			Timestamp: uint32(time.Now().Unix()),
		},
	}

	rollermanager.InitRollerManager()
	pubKey, err := tmpAuthMsg.PublicKey()
	assert.NoError(t, err)

	taskChan, err := rollermanager.Manager.Register(pubKey, tmpAuthMsg.Identity)
	assert.NotNil(t, taskChan)
	assert.NoError(t, err)

	convey.Convey("verify failure", t, func() {
		var s *message.ProofMsg
		patchGuard := gomonkey.ApplyMethodFunc(s, "Verify", func() (bool, error) {
			return false, errors.New("proof verify error")
		})
		defer patchGuard.Reset()
		err := rollerController.SubmitProof(tmpProof)
		assert.Error(t, err)
	})

	var s *message.ProofMsg
	patchGuard := gomonkey.ApplyMethodFunc(s, "Verify", func() (bool, error) {
		return true, nil
	})
	defer patchGuard.Reset()

	convey.Convey("get rollers info failure", t, func() {
		err := rollerController.SubmitProof(tmpProof)
		targetErr := fmt.Errorf("proof generation session for id %v does not existID", tmpProof.ID)
		assert.Equal(t, err.Error(), targetErr.Error())
	})

	rollerStatusMap := make(map[string]*coordinatorType.RollerStatus)
	rollerStatusMap[pubKey] = &coordinatorType.RollerStatus{
		PublicKey: pubKey,
		Name:      "test-submit-proof-roller",
		Status:    types.RollerProofValid,
	}

	rollersInfo := &coordinatorType.RollersInfo{
		ID:             "rollers_info_test",
		Rollers:        rollerStatusMap,
		StartTimestamp: time.Now().Unix(),
		ProveType:      message.BasicProve,
	}
	rollermanager.Manager.AddRollerInfo(rollersInfo)

	convey.Convey("get none rollers of rollersInfo", t, func() {
		tmpProof1 := &message.ProofMsg{
			ProofDetail: &message.ProofDetail{
				ID:     "10001",
				Status: message.StatusOk,
				Proof:  &message.AggProof{},
			},
		}
		pubKey1, err1 := tmpProof1.PublicKey()
		assert.NoError(t, err1)
		err := rollerController.SubmitProof(tmpProof1)
		targetErr := fmt.Errorf("get none rollers for the proof key:%s id:%s", pubKey1, tmpProof1.ID)
		assert.Equal(t, err.Error(), targetErr.Error())
	})

	convey.Convey("roller status is RollerProofValid", t, func() {
		err1 := rollerController.SubmitProof(tmpProof)
		targetErr := fmt.Errorf("roller has already submitted valid proof in proof session")
		assert.Contains(t, err1.Error(), targetErr.Error())
	})
	rollerStatusMap[pubKey].Status = types.RollerAssigned

	convey.Convey("proof msg status is not ok", t, func() {
		tmpProof.Status = message.StatusProofError
		err1 := rollerController.SubmitProof(tmpProof)
		assert.Nil(t, err1)
	})
	tmpProof.Status = message.StatusOk

	var blockBatchOrm *orm.BlockBatch
	convey.Convey("basic prove store proof content failure", t, func() {
		targetError := errors.New("UpdateProofAndHashByHash error")
		patchGuard.ApplyMethodFunc(blockBatchOrm, "UpdateProofAndHashByHash", func(context.Context, string, []byte, uint64, types.ProvingStatus) error {
			return targetError
		})
		err1 := rollerController.SubmitProof(tmpProof)
		assert.Equal(t, err1.Error(), targetError.Error())
	})

	var aggTaskOrm *orm.AggTask
	convey.Convey("agg prove store proof content failure", t, func() {
		tmpProof.Type = message.AggregatorProve
		targetError := errors.New("UpdateProofForAggTask error")
		patchGuard.ApplyMethodFunc(aggTaskOrm, "UpdateProofForAggTask", func(aggTaskID string, proof []byte) error {
			return targetError
		})
		err1 := rollerController.SubmitProof(tmpProof)
		assert.Equal(t, err1.Error(), targetError.Error())
	})

	tmpProof.Type = message.BasicProve
	patchGuard.ApplyMethodFunc(blockBatchOrm, "UpdateProofAndHashByHash", func(context.Context, string, []byte, uint64, types.ProvingStatus) error {
		return nil
	})

	var tmpVerifier *verifier.Verifier
	convey.Convey("verifier proof failure", t, func() {
		targetErr := errors.New("verify proof failure")
		patchGuard.ApplyMethodFunc(tmpVerifier, "VerifyProof", func(proof *message.AggProof) (bool, error) {
			return false, targetErr
		})
		err1 := rollerController.SubmitProof(tmpProof)
		assert.Nil(t, err1)
	})

	patchGuard.ApplyMethodFunc(tmpVerifier, "VerifyProof", func(proof *message.AggProof) (bool, error) {
		return true, nil
	})

	convey.Convey("closeProofTask BasicProve update status failure", t, func() {
		targetError := errors.New("UpdateProvingStatus error")
		patchGuard.ApplyMethodFunc(blockBatchOrm, "UpdateProvingStatus", func(hash string, status types.ProvingStatus) error {
			return targetError
		})
		err1 := rollerController.SubmitProof(tmpProof)
		assert.Equal(t, err1.Error(), targetError.Error())
	})

	convey.Convey("closeProofTask AggregatorProve update status failure", t, func() {
		tmpProof.Type = message.AggregatorProve
		targetError := errors.New("UpdateAggTaskStatus error")
		patchGuard.ApplyMethodFunc(aggTaskOrm, "UpdateAggTaskStatus", func(aggTaskID string, status types.ProvingStatus) error {
			return targetError
		})
		err1 := rollerController.SubmitProof(tmpProof)
		assert.Equal(t, err1.Error(), targetError.Error())
	})

	tmpProof.Type = message.BasicProve
	patchGuard.ApplyMethodFunc(blockBatchOrm, "UpdateProvingStatus", func(hash string, status types.ProvingStatus) error {
		return nil
	})

	err1 := rollerController.SubmitProof(tmpProof)
	assert.Nil(t, err1)
}
