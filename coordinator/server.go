package coordinator

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"scroll-tech/common/message"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/log"
)

const (
	// HandshakeTimeout is a time limit for a handshake to succeed.
	HandshakeTimeout = 10 * time.Second

	// The amount of time it's acceptable to wait for a pong message.
	pongWait = 60 * time.Second
	// The interval between which we send pings.
	pingWait  = (pongWait / 10) * 9
	writeWait = 10 * time.Second

	wsMessageSizeLimit = 150 * 1024 * 1024 // 150 MB
)

var upgrader = websocket.Upgrader{} // use default options

// Roller represents a websocket connection to a roller, and includes
// the roller authentication, and the websocket connection to the roller.
type Roller struct {
	AuthMsg *message.AuthMessage

	// A mutex guarding the ws write methods.
	wMu sync.RWMutex
	ws  *websocket.Conn

	closed  int64
	closeCh chan struct{}
}

func (r *Roller) close() error {
	if r.isClosed() {
		return nil
	}
	atomic.StoreInt64(&r.closed, 1)
	close(r.closeCh)
	if err := r.ws.Close(); err != nil {
		log.Error("fail to close WS", "err", err)
		return err
	}
	return nil
}

func (r *Roller) isClosed() bool {
	return atomic.LoadInt64(&r.closed) != 0
}

// A convenient struct to send over incoming messages coupled with their
// associated roller pk to the roller manager.
type messageAndPk struct {
	pk      string
	message []byte
}

// A websocket server, responsible for establishing connections with rollers,
// and reading their messages, before passing them on to be handled by the roller manager.
type server struct {
	rollerChan chan *Roller
	server     *http.Server

	// All live connections to the rollers in the network.
	conns *conns
	// Channel to send incoming messages to (goes to the roller manager).
	msgChan chan *messageAndPk
}

func newServer(addr string) *server {
	s := &server{
		rollerChan: make(chan *Roller, 100),
		conns:      newConns(),
		msgChan:    make(chan *messageAndPk, 100),
	}

	var srv http.Server
	s.server = &srv
	s.server.Addr = addr
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.wsHandler)
	s.server.Handler = mux

	return s
}

func (s *server) start() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var err error
	go func() {
		err = s.server.ListenAndServe()
	}()

	<-ctx.Done()
	return err
}

func (s *server) stop() error {
	s.conns.clear()
	return s.server.Shutdown(context.Background())
}

func (s *server) wsHandler(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Debug("Could not upgrade", "error", err)
		return
	}
	c.SetReadLimit(wsMessageSizeLimit)

	// There will not be concurrent read/write on a connection when we handshake
	// at the very beginning, so there's no need to lock here.
	authMessage, err := s.handshake(c)
	if err != nil {
		log.Error("Could not complete handshake", "error", err)
		return
	}

	roller := &Roller{
		AuthMsg: authMessage,
		ws:      c,
		closeCh: make(chan struct{}),
	}

	// Overwrite existing connection.
	// We don't need to worry about a malicious roller faking its pubkey, because
	// we've checked `VerifySignature` in `handshake` above
	if s.conns.get(roller.AuthMsg.Identity.PublicKey) != nil {
		log.Warn("Roller attempted to connect more than once",
			"name", roller.AuthMsg.Identity.Name,
			"pk", roller.AuthMsg.Identity.PublicKey)
	}

	// There will not be concurrent read/write on a connection when we
	// SetPingHandler/SetPongHandler at the very beginning, so there's no need
	// to lock for them. But we may still need to lock for the handlers.
	roller.ws.SetPingHandler(
		func(string) error {
			roller.wMu.Lock()
			defer roller.wMu.Unlock()
			return roller.ws.WriteMessage(websocket.PongMessage, nil)
		})
	roller.ws.SetPongHandler(
		func(string) error {
			return roller.ws.SetReadDeadline(time.Now().Add(pongWait))
		})
	s.conns.add(roller)
	go s.readLoop(roller)
	go s.pingLoop(roller)

	// avoid being blocked
	if s.rollerChan != nil {
		select {
		case s.rollerChan <- roller:
		default:
			return
		}
	}
}

func (s *server) handshake(c *websocket.Conn) (*message.AuthMessage, error) {
	// Set up a timer so that we won't be left hanging by an unresponsive roller.
	t := time.AfterFunc(HandshakeTimeout, func() {
		_ = c.Close()
	})

	// We expect an authentication message to come in from the roller.

	payload, err := func(c *websocket.Conn) ([]byte, error) {
		for {
			mt, payload, err := c.ReadMessage()
			if err != nil {
				return nil, err
			}

			if mt == websocket.BinaryMessage {
				return payload, nil
			}
		}
	}(c)
	if err != nil {
		return nil, err
	}

	// Read succeeded, cancel timer before we accidentally close the connection.
	t.Stop()

	msg := &message.Msg{}
	if err = json.Unmarshal(payload, msg); err != nil {
		return nil, err
	}

	// We should receive a Register message
	if msg.Type != message.RegisterMsgType {
		return nil, errors.New("got wrong handshake message, expected Register")
	}

	authMsg := &message.AuthMessage{}
	if err = json.Unmarshal(msg.Payload, authMsg); err != nil {
		return nil, err
	}

	// Verify signature
	hash, err := authMsg.Identity.Hash()
	if err != nil {
		return nil, err
	}

	if !crypto.VerifySignature(common.FromHex(authMsg.Identity.PublicKey), hash, common.FromHex(authMsg.Signature)[:64]) {
		return nil, errors.New("signature verification failed")
	}
	log.Info("signature verification successfully", "roller name", authMsg.Identity.Name)

	return authMsg, nil
}

func (s *server) readLoop(c *Roller) {
	defer s.conns.delete(c)

	for {
		select {
		case <-c.closeCh:
			return
		default:
			if err := c.ws.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
				log.Error("could not set read deadline", "error", err)
				return
			}

			mt, msg, err := c.ws.ReadMessage()
			if err != nil {
				log.Error("could not read msg", "error", err, "name", c.AuthMsg.Identity.Name)
				return
			}

			// Check if this msg needs to be handled manually.
			if mt == websocket.BinaryMessage {
				log.Trace("websocket msg received", "msg", msg)

				s.msgChan <- &messageAndPk{
					pk:      c.AuthMsg.Identity.PublicKey,
					message: msg,
				}
			}
		}
	}
}

// Run a ping loop.
func (s *server) pingLoop(c *Roller) {
	pingTicker := time.NewTicker(pingWait)
	defer pingTicker.Stop()

	for {
		select {
		case <-c.closeCh:
			return
		case <-pingTicker.C:
			c.wMu.Lock()

			if err := c.ws.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Error("could not set write deadline", "error", err)
				c.wMu.Unlock()
				return
			}

			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Error("could not send ping", "error", err)
				c.wMu.Unlock()
				return
			}

			c.wMu.Unlock()
		}
	}
}

func (r *Roller) sendMessage(msg message.Msg) error {
	b, err := json.Marshal(&msg)
	if err != nil {
		return err
	}

	r.wMu.Lock()
	defer r.wMu.Unlock()

	if err := r.ws.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		log.Error("could not set write deadline", "error", err)
		return err
	}
	return r.ws.WriteMessage(websocket.BinaryMessage, b)
}
