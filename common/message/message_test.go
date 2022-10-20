package message

import (
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

func TestAuthMessageSignAndVerify(t *testing.T) {
	privkey, err := crypto.GenerateKey()
	assert.NoError(t, err)

	authMsg := &AuthMessage{
		Identity: &Identity{
			Name:      "testRoller",
			Timestamp: time.Now().UnixNano(),
		},
	}
	assert.NoError(t, authMsg.Sign(privkey))

	ok, err := authMsg.Verify()
	assert.NoError(t, err)
	assert.Equal(t, true, ok)
}
