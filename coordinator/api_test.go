package coordinator

import (
	"context"
	"errors"
	"testing"
	"time"

	sm "github.com/cch123/supermonkey"
	cmap "github.com/orcaman/concurrent-map"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/message"

	"scroll-tech/coordinator/config"
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
	assert.NoError(t, authMsg.Sign(privKey))
	return authMsg
}

var rollerManager *Manager

func TestMain(m *testing.M) {
	initRollerManager()
	m.Run()
}

func initRollerManager() {
	rmConfig := config.RollerManagerConfig{}
	rmConfig.Verifier = &config.VerifierConfig{MockMode: true}
	rollerManager, _ = New(context.Background(), &rmConfig, nil, nil)
}

func TestManager_RequestToken(t *testing.T) {
	tmpAuthMsg := &message.AuthMsg{
		Identity: &message.Identity{
			Name:      "roller_test_request_token",
			Timestamp: uint32(time.Now().Unix()),
		},
	}

	tokenCacheStored := "c393987bb791dd285dd3d8ffbd770ed1"

	convey.Convey("auth msg verify failure", t, func() {
		token, err := rollerManager.RequestToken(tmpAuthMsg)
		assert.Error(t, err)
		assert.Empty(t, token)
	})

	convey.Convey("token get failure", t, func() {
		tmpAuthMsg := geneAuthMsg(t)
		patchGuard := sm.PatchByFullSymbolName("github.com/patrickmn/go-cache.(*cache).Get", func(ptr uintptr, abc string) (interface{}, bool) {
			return tokenCacheStored, true
		})
		token, err := rollerManager.RequestToken(tmpAuthMsg)
		assert.NoError(t, err)
		assert.Equal(t, token, tokenCacheStored)
		patchGuard.Unpatch()
	})

	convey.Convey("token generation failure", t, func() {
		tmpAuthMsg = geneAuthMsg(t)
		patchGuard := sm.Patch(message.GenerateToken, func() (string, error) {
			return "", errors.New("token generation failed")
		})
		token, err := rollerManager.RequestToken(tmpAuthMsg)
		assert.Error(t, err)
		assert.Empty(t, token)
		patchGuard.Unpatch()
	})

	convey.Convey("token generation success", t, func() {
		tmpAuthMsg = geneAuthMsg(t)
		patchGuard := sm.Patch(message.GenerateToken, func() (string, error) {
			return tokenCacheStored, nil
		})
		token, err := rollerManager.RequestToken(tmpAuthMsg)
		assert.NoError(t, err)
		assert.Equal(t, tokenCacheStored, token)
		patchGuard.Unpatch()
	})
}

func TestManager_Register(t *testing.T) {
	tmpAuthMsg := &message.AuthMsg{
		Identity: &message.Identity{
			Name:      "roller_test_register",
			Timestamp: uint32(time.Now().Unix()),
		},
	}

	convey.Convey("auth msg verify failure", t, func() {
		subscription, err := rollerManager.Register(context.Background(), tmpAuthMsg)
		assert.Error(t, err)
		assert.Empty(t, subscription)
	})

	convey.Convey("verify token failure", t, func() {
		tmpAuthMsg = geneAuthMsg(t)
		patchGuard7 := sm.Patch((*Manager).VerifyToken, func(manager *Manager, tmpAuthMsg *message.AuthMsg) (bool, error) {
			return false, errors.New("verify token failure")
		})
		subscription, err := rollerManager.Register(context.Background(), tmpAuthMsg)
		assert.Error(t, err)
		assert.Empty(t, subscription)
		patchGuard7.Unpatch()
	})

	convey.Convey("notifier failure", t, func() {
		tmpAuthMsg = geneAuthMsg(t)
		patchGuard7 := sm.Patch((*Manager).VerifyToken, func(manager *Manager, tmpAuthMsg *message.AuthMsg) (bool, error) {
			return true, nil
		})
		patchGuard8 := sm.Patch(rpc.NotifierFromContext, func(ctx context.Context) (*rpc.Notifier, bool) {
			return nil, false
		})
		subscription, err := rollerManager.Register(context.Background(), tmpAuthMsg)
		assert.Error(t, err)
		assert.Equal(t, err, rpc.ErrNotificationsUnsupported)
		assert.Equal(t, *subscription, rpc.Subscription{})
		patchGuard8.Unpatch()
		patchGuard7.Unpatch()
	})
}

func TestManager_SubmitProof(t *testing.T) {
	id := "10000"
	proof := &message.ProofMsg{
		ProofDetail: &message.ProofDetail{
			ID:     id,
			Status: message.StatusOk,
			Proof:  &message.AggProof{},
		},
	}

	var rp rollerNode
	rp.TaskIDs = cmap.New()
	rp.TaskIDs.Set(id, id)

	convey.Convey("verify failure", t, func() {
		patchGuard := sm.Patch((*message.ProofMsg).Verify, func(*message.ProofMsg) (bool, error) {
			return false, errors.New("proof verify error")
		})
		isSuccess, err := rollerManager.SubmitProof(proof)
		assert.False(t, isSuccess)
		assert.Error(t, err)
		patchGuard.Unpatch()
	})

	convey.Convey("existTaskIDForRoller failure", t, func() {
		patchGuard1 := sm.PatchByFullSymbolName("github.com/orcaman/concurrent-map.(*ConcurrentMap).Get", func(ptr uintptr, key string) (interface{}, bool) {
			return nil, true
		})
		patchGuard2 := sm.Patch((*message.ProofMsg).Verify, func(*message.ProofMsg) (bool, error) {
			return true, nil
		})
		isSuccess, err := rollerManager.SubmitProof(proof)
		assert.False(t, isSuccess)
		assert.Error(t, err)
		patchGuard2.Unpatch()
		patchGuard1.Unpatch()
	})

	convey.Convey("handleZkProof failure", t, func() {
		patchGuard51 := sm.Patch((*message.ProofMsg).Verify, func(*message.ProofMsg) (bool, error) {
			return true, nil
		})
		patchGuard61 := sm.PatchByFullSymbolName("github.com/orcaman/concurrent-map.ConcurrentMap.Get", func(ptr uintptr, key string) (interface{}, bool) {
			return &rp, true
		})
		patchGuard3 := sm.Patch((*Manager).handleZkProof, func(manager *Manager, pk string, msg *message.ProofDetail) error {
			return errors.New("handle zk proof error")
		})
		isSuccess, err := rollerManager.SubmitProof(proof)
		assert.Error(t, err)
		assert.False(t, isSuccess)
		patchGuard3.Unpatch()
		patchGuard51.Unpatch()
		patchGuard61.Unpatch()
	})

	convey.Convey("SubmitProof success", t, func() {
		patchGuard5 := sm.Patch((*message.ProofMsg).Verify, func(*message.ProofMsg) (bool, error) {
			return true, nil
		})
		patchGuard6 := sm.PatchByFullSymbolName("github.com/orcaman/concurrent-map.ConcurrentMap.Get", func(ptr uintptr, key string) (interface{}, bool) {
			return &rp, true
		})
		patchGuard7 := sm.Patch((*Manager).handleZkProof, func(manager *Manager, pk string, msg *message.ProofDetail) error {
			return nil
		})
		isSuccess, err := rollerManager.SubmitProof(proof)
		assert.NoError(t, err)
		assert.True(t, isSuccess)
		patchGuard5.Unpatch()
		patchGuard6.Unpatch()
		patchGuard7.Unpatch()
	})
}
