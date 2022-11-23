package mockroller

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"scroll-tech/common/message"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/gorilla/websocket"
)

// MockRoller is mock roller for coordinator unit tests
type MockRoller struct {
	PublicKey  []byte
	PrivateKey []byte
	Sessions   map[string]bool
}

func mustGenerateKeyPair() (pubkey, privkey []byte) {
	key, err := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	pubkey = elliptic.Marshal(secp256k1.S256(), key.X, key.Y)

	privkey = make([]byte, 32)
	blob := key.D.Bytes()
	copy(privkey[32-len(blob):], blob)

	return pubkey, privkey
}

// MustNewmockRoller generates a pair of public key and private key for the mock roller
func MustNewmockRoller() MockRoller {
	publicKey, privateKey := mustGenerateKeyPair()
	return MockRoller{PublicKey: publicKey, PrivateKey: privateKey}
}

// PerformHandshake performs handshake with the coordinator
func (r *MockRoller) PerformHandshake(c *websocket.Conn) error {
	authMsg := &message.AuthMessage{
		Identity: message.Identity{
			Name:      "testRoller-" + common.Bytes2Hex(r.PublicKey),
			Timestamp: time.Now().UnixNano(),
			PublicKey: common.Bytes2Hex(r.PublicKey),
		},
		Signature: "",
	}
	hash, err := authMsg.Identity.Hash()
	if err != nil {
		return err
	}
	sig, err := secp256k1.Sign(hash, r.PrivateKey)
	if err != nil {
		return err
	}
	authMsg.Signature = common.Bytes2Hex(sig)
	b, err := json.Marshal(authMsg)
	if err != nil {
		return err
	}
	msg := &message.Msg{
		Type:    message.RegisterMsgType,
		Payload: b,
	}
	b, err = json.Marshal(msg)
	if err != nil {
		return err
	}
	err = c.WriteMessage(websocket.BinaryMessage, b)
	return err
}
