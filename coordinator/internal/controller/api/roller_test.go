package api

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/proof"
	"scroll-tech/coordinator/internal/logic/rollermanager"
	"scroll-tech/coordinator/internal/logic/verifier"
	"scroll-tech/coordinator/internal/orm"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

func geneAuthMsg(t *testing.T) (*message.AuthMsg, *ecdsa.PrivateKey) {
	authMsg := &message.AuthMsg{
		Identity: &message.Identity{
			Name: "roller_test1",
		},
	}
	privKey, err := crypto.GenerateKey()
	assert.NoError(t, err)
	assert.NoError(t, authMsg.SignWithKey(privKey))
	return authMsg, privKey
}

var rollerController *RollerController

func init() {
	conf := &config.RollerManagerConfig{
		TokenTimeToLive: 120,
	}
	conf.Verifier = &config.VerifierConfig{MockMode: true}
	rollerController = NewRollerController(conf, nil)
}

func TestRoller_RequestToken(t *testing.T) {
	convey.Convey("auth msg verify failure", t, func() {
		tmpAuthMsg := &message.AuthMsg{
			Identity: &message.Identity{
				Name: "roller_test_request_token",
			},
		}
		token, err := rollerController.RequestToken(tmpAuthMsg)
		assert.Error(t, err)
		assert.Empty(t, token)
	})

	convey.Convey("token has already been distributed", t, func() {
		tmpAuthMsg, _ := geneAuthMsg(t)
		key, _ := tmpAuthMsg.PublicKey()
		tokenCacheStored := "c393987bb791dd285dd3d8ffbd770ed1"
		rollerController.tokenCache.Set(key, tokenCacheStored, time.Hour)
		token, err := rollerController.RequestToken(tmpAuthMsg)
		assert.NoError(t, err)
		assert.Equal(t, token, tokenCacheStored)
	})

	convey.Convey("token generation failure", t, func() {
		tmpAuthMsg, _ := geneAuthMsg(t)
		patchGuard := gomonkey.ApplyFunc(message.GenerateToken, func() (string, error) {
			return "", errors.New("token generation failed")
		})
		defer patchGuard.Reset()
		token, err := rollerController.RequestToken(tmpAuthMsg)
		assert.Error(t, err)
		assert.Empty(t, token)
	})

	convey.Convey("token generation success", t, func() {
		tmpAuthMsg, _ := geneAuthMsg(t)
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

func TestRoller_Register(t *testing.T) {
	convey.Convey("auth msg verify failure", t, func() {
		tmpAuthMsg := &message.AuthMsg{
			Identity: &message.Identity{
				Name: "roller_test_register",
			},
		}
		subscription, err := rollerController.Register(context.Background(), tmpAuthMsg)
		assert.Error(t, err)
		assert.Empty(t, subscription)
	})

	convey.Convey("verify token failure", t, func() {
		tmpAuthMsg, _ := geneAuthMsg(t)
		patchGuard := gomonkey.ApplyPrivateMethod(rollerController, "verifyToken", func(tmpAuthMsg *message.AuthMsg) (bool, error) {
			return false, errors.New("verify token failure")
		})
		defer patchGuard.Reset()
		subscription, err := rollerController.Register(context.Background(), tmpAuthMsg)
		assert.Error(t, err)
		assert.Empty(t, subscription)
	})

	convey.Convey("notifier failure", t, func() {
		tmpAuthMsg, _ := geneAuthMsg(t)
		patchGuard := gomonkey.ApplyPrivateMethod(rollerController, "verifyToken", func(tmpAuthMsg *message.AuthMsg) (bool, error) {
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
		tmpAuthMsg, _ := geneAuthMsg(t)
		patchGuard := gomonkey.ApplyPrivateMethod(rollerController, "verifyToken", func(tmpAuthMsg *message.AuthMsg) (bool, error) {
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
		tmpAuthMsg, _ := geneAuthMsg(t)
		patchGuard := gomonkey.ApplyPrivateMethod(rollerController, "verifyToken", func(tmpAuthMsg *message.AuthMsg) (bool, error) {
			return true, nil
		})
		defer patchGuard.Reset()

		var taskWorker *proof.TaskWorker
		patchGuard.ApplyPrivateMethod(taskWorker, "AllocTaskWorker", func(ctx context.Context, authMsg *message.AuthMsg) (*rpc.Subscription, error) {
			return nil, nil
		})
		_, err := rollerController.Register(context.Background(), tmpAuthMsg)
		assert.NoError(t, err)
	})
}

func TestRoller_SubmitProof(t *testing.T) {
	tmpAuthMsg, prvKey := geneAuthMsg(t)
	pubKey, err := tmpAuthMsg.PublicKey()
	assert.NoError(t, err)

	id := "rollers_info_test"
	tmpProof := &message.ProofMsg{
		ProofDetail: &message.ProofDetail{
			Type:   message.ProofTypeChunk,
			ID:     id,
			Status: message.StatusOk,
			Proof:  &message.AggProof{},
		},
	}
	assert.NoError(t, tmpProof.Sign(prvKey))
	proofPubKey, err := tmpProof.PublicKey()
	assert.NoError(t, err)
	assert.Equal(t, pubKey, proofPubKey)

	rollermanager.InitRollerManager()

	taskChan, err := rollermanager.Manager.Register(pubKey, tmpAuthMsg.Identity)
	assert.NotNil(t, taskChan)
	assert.NoError(t, err)

	convey.Convey("verify failure", t, func() {
		var s *message.ProofMsg
		patchGuard := gomonkey.ApplyMethodFunc(s, "Verify", func() (bool, error) {
			return false, errors.New("proof verify error")
		})
		defer patchGuard.Reset()
		err = rollerController.SubmitProof(tmpProof)
		assert.Error(t, err)
	})

	var s *message.ProofMsg
	patchGuard := gomonkey.ApplyMethodFunc(s, "Verify", func() (bool, error) {
		return true, nil
	})
	defer patchGuard.Reset()

	var chunkOrm *orm.Chunk
	patchGuard.ApplyMethodFunc(chunkOrm, "UpdateProofByHash", func(context.Context, string, *message.AggProof, uint64, ...*gorm.DB) error {
		return nil
	})
	patchGuard.ApplyMethodFunc(chunkOrm, "UpdateProvingStatus", func(ctx context.Context, hash string, status types.ProvingStatus, dbTX ...*gorm.DB) error {
		return nil
	})

	var batchOrm *orm.Batch
	patchGuard.ApplyMethodFunc(batchOrm, "UpdateProofByHash", func(ctx context.Context, hash string, proof *message.AggProof, proofTimeSec uint64, dbTX ...*gorm.DB) error {
		return nil
	})
	patchGuard.ApplyMethodFunc(batchOrm, "UpdateProvingStatus", func(ctx context.Context, hash string, status types.ProvingStatus, dbTX ...*gorm.DB) error {
		return nil
	})

	var proverTaskOrm *orm.ProverTask
	convey.Convey("get none rollers of prover task", t, func() {
		patchGuard.ApplyMethodFunc(proverTaskOrm, "GetProverTaskByHashAndPubKey", func(ctx context.Context, hash, pubKey string) (*orm.ProverTask, error) {
			return nil, nil
		})
		tmpProof1 := &message.ProofMsg{
			ProofDetail: &message.ProofDetail{
				ID:     "10001",
				Status: message.StatusOk,
				Proof:  &message.AggProof{},
			},
		}
		privKey, err := crypto.GenerateKey()
		assert.NoError(t, err)
		tmpProof1.Sign(privKey)
		_, err1 := tmpProof1.PublicKey()
		assert.NoError(t, err1)
		err2 := rollerController.SubmitProof(tmpProof1)
		fmt.Println(err2)
		targetErr := fmt.Errorf("validator failure get none rollers for the proof")
		assert.Equal(t, err2.Error(), targetErr.Error())
	})

	patchGuard.ApplyMethodFunc(proverTaskOrm, "GetProverTaskByHashAndPubKey", func(ctx context.Context, hash, pubKey string) (*orm.ProverTask, error) {
		now := time.Now()
		s := &orm.ProverTask{
			TaskID:          id,
			ProverPublicKey: proofPubKey,
			TaskType:        int16(message.ProofTypeChunk),
			ProverName:      "rollers_info_test",
			ProvingStatus:   int16(types.RollerAssigned),
			CreatedAt:       now,
		}
		return s, nil
	})

	patchGuard.ApplyMethodFunc(proverTaskOrm, "UpdateProverTaskProvingStatus", func(ctx context.Context, proofType message.ProofType, taskID string, pk string, status types.RollerProveStatus, dbTX ...*gorm.DB) error {
		return nil
	})

	patchGuard.ApplyPrivateMethod(rollerController.proofReceiver, "proofFailure", func(hash string, pubKey string, proofMsgType message.ProofType) {
	})

	convey.Convey("proof msg status is not ok", t, func() {
		tmpProof.Status = message.StatusProofError
		err1 := rollerController.SubmitProof(tmpProof)
		assert.NoError(t, err1)
	})
	tmpProof.Status = message.StatusOk

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

	patchGuard.ApplyMethodFunc(tmpVerifier, "VerifyProof", func(proof *message.AggProof) (bool, error) {
		return true, nil
	})

	patchGuard.ApplyPrivateMethod(rollerController.proofReceiver, "closeProofTask", func(hash string, pubKey string, proofMsg *message.ProofMsg, rollersInfo *coordinatorType.RollersInfo) error {
		return nil
	})

	err1 := rollerController.SubmitProof(tmpProof)
	assert.Nil(t, err1)
}
