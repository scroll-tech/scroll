package rollers

import (
	cmap "github.com/orcaman/concurrent-map"
	"github.com/scroll-tech/go-ethereum/log"
)

// A thread-safe wrapper for a connection list.
type conns struct {
	cm cmap.ConcurrentMap
}

func newConns() *conns {
	return &conns{
		cm: cmap.New(),
	}
}

func (c *conns) add(conn *Roller) {
	hexPk := conn.AuthMsg.Identity.PublicKey
	_ = c.cm.Upsert(hexPk, conn, swapFn)
}

func swapFn(exists bool, valueInMap interface{}, newValue interface{}) interface{} {
	// If the roller already exists, close its connection.
	if exists {
		_ = valueInMap.(*Roller).ws.Close()
	}

	return newValue
}

func (c *conns) get(pk string) *Roller {
	roller, ok := c.cm.Get(pk)
	if ok {
		return roller.(*Roller)
	}

	return nil
}

func (c *conns) delete(conn *Roller) {
	if err := conn.close(); err != nil {
		log.Error("failed to close ws handler", "name", conn.AuthMsg.Identity.Name, "error", err)
	}
	hexPk := conn.AuthMsg.Identity.PublicKey
	_ = c.cm.RemoveCb(hexPk, removeFn)
}

func removeFn(key string, v interface{}, exists bool) bool {
	if exists {
		_ = v.(*Roller).ws.Close()
	}

	return true
}

func (c *conns) clear() {
	for tuple := range c.cm.IterBuffered() {
		c.delete(tuple.Val.(*Roller))
	}
}

func (c *conns) getAll() (allConns []*Roller) {
	for tuple := range c.cm.IterBuffered() {
		allConns = append(allConns, tuple.Val.(*Roller))
	}
	return allConns
}
