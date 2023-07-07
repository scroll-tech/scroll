package coordinator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	cmap "github.com/orcaman/concurrent-map"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
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

var rollerManager *Manager

func init() {
	rmConfig := config.RollerManagerConfig{}
	rmConfig.Verifier = &config.VerifierConfig{MockMode: true}
	rollerManager, _ = New(context.Background(), &rmConfig, nil)
}

func TestManager_RequestToken(t *testing.T) {
	convey.Convey("auth msg verify failure", t, func() {
		tmpAuthMsg := &message.AuthMsg{
			Identity: &message.Identity{
				Name:      "roller_test_request_token",
				Timestamp: uint32(time.Now().Unix()),
			},
		}
		token, err := rollerManager.RequestToken(tmpAuthMsg)
		assert.Error(t, err)
		assert.Empty(t, token)
	})

	convey.Convey("token has already been distributed", t, func() {
		tmpAuthMsg := geneAuthMsg(t)
		key, _ := tmpAuthMsg.PublicKey()
		tokenCacheStored := "c393987bb791dd285dd3d8ffbd770ed1"
		rollerManager.tokenCache.Set(key, tokenCacheStored, time.Hour)
		token, err := rollerManager.RequestToken(tmpAuthMsg)
		assert.NoError(t, err)
		assert.Equal(t, token, tokenCacheStored)
	})

	convey.Convey("token generation failure", t, func() {
		tmpAuthMsg := geneAuthMsg(t)
		patchGuard := gomonkey.ApplyFunc(message.GenerateToken, func() (string, error) {
			return "", errors.New("token generation failed")
		})
		defer patchGuard.Reset()
		token, err := rollerManager.RequestToken(tmpAuthMsg)
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
		token, err := rollerManager.RequestToken(tmpAuthMsg)
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
		subscription, err := rollerManager.Register(context.Background(), tmpAuthMsg)
		assert.Error(t, err)
		assert.Empty(t, subscription)
	})

	convey.Convey("verify token failure", t, func() {
		tmpAuthMsg := geneAuthMsg(t)
		patchGuard := gomonkey.ApplyMethodFunc(rollerManager, "VerifyToken", func(tmpAuthMsg *message.AuthMsg) (bool, error) {
			return false, errors.New("verify token failure")
		})
		defer patchGuard.Reset()
		subscription, err := rollerManager.Register(context.Background(), tmpAuthMsg)
		assert.Error(t, err)
		assert.Empty(t, subscription)
	})

	convey.Convey("register failure", t, func() {
		tmpAuthMsg := geneAuthMsg(t)
		patchGuard := gomonkey.ApplyMethodFunc(rollerManager, "VerifyToken", func(tmpAuthMsg *message.AuthMsg) (bool, error) {
			return true, nil
		})
		defer patchGuard.Reset()
		patchGuard.ApplyPrivateMethod(rollerManager, "register", func(*Manager, string, *message.Identity) (<-chan *message.TaskMsg, error) {
			return nil, errors.New("register error")
		})
		subscription, err := rollerManager.Register(context.Background(), tmpAuthMsg)
		assert.Error(t, err)
		assert.Empty(t, subscription)
	})

	convey.Convey("notifier failure", t, func() {
		tmpAuthMsg := geneAuthMsg(t)
		patchGuard := gomonkey.ApplyMethodFunc(rollerManager, "VerifyToken", func(tmpAuthMsg *message.AuthMsg) (bool, error) {
			return true, nil
		})
		defer patchGuard.Reset()
		patchGuard.ApplyFunc(rpc.NotifierFromContext, func(ctx context.Context) (*rpc.Notifier, bool) {
			return nil, false
		})
		subscription, err := rollerManager.Register(context.Background(), tmpAuthMsg)
		assert.Error(t, err)
		assert.Equal(t, err, rpc.ErrNotificationsUnsupported)
		assert.Equal(t, *subscription, rpc.Subscription{})
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
		var s *message.ProofMsg
		patchGuard := gomonkey.ApplyMethodFunc(s, "Verify", func() (bool, error) {
			return false, errors.New("proof verify error")
		})
		defer patchGuard.Reset()
		err := rollerManager.SubmitProof(proof)
		assert.Error(t, err)
	})

	convey.Convey("existTaskIDForRoller failure", t, func() {
		var s *cmap.ConcurrentMap
		patchGuard := gomonkey.ApplyMethodFunc(s, "Get", func(key string) (interface{}, bool) {
			return nil, true
		})
		defer patchGuard.Reset()

		var pm *message.ProofMsg
		patchGuard.ApplyMethodFunc(pm, "Verify", func() (bool, error) {
			return true, nil
		})
		err := rollerManager.SubmitProof(proof)
		assert.Error(t, err)
	})

	convey.Convey("handleZkProof failure", t, func() {
		var pm *message.ProofMsg
		patchGuard := gomonkey.ApplyMethodFunc(pm, "Verify", func() (bool, error) {
			return true, nil
		})
		defer patchGuard.Reset()

		var s cmap.ConcurrentMap
		patchGuard.ApplyMethodFunc(s, "Get", func(key string) (interface{}, bool) {
			return &rp, true
		})

		patchGuard.ApplyPrivateMethod(rollerManager, "handleZkProof", func(manager *Manager, pk string, msg *message.ProofDetail) error {
			return errors.New("handle zk proof error")
		})

		err := rollerManager.SubmitProof(proof)
		assert.Error(t, err)
	})

	convey.Convey("SubmitProof success", t, func() {
		var pm *message.ProofMsg
		patchGuard := gomonkey.ApplyMethodFunc(pm, "Verify", func() (bool, error) {
			return true, nil
		})
		defer patchGuard.Reset()

		var s cmap.ConcurrentMap
		patchGuard.ApplyMethodFunc(s, "Get", func(key string) (interface{}, bool) {
			return &rp, true
		})

		patchGuard.ApplyPrivateMethod(rollerManager, "handleZkProof", func(manager *Manager, pk string, msg *message.ProofDetail) error {
			return nil
		})

		err := rollerManager.SubmitProof(proof)
		assert.NoError(t, err)
	})
}
