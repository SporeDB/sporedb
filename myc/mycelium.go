// Package myc contains the SporeDB mycelium logic.
package myc

import (
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"

	lru "github.com/hashicorp/golang-lru"

	"gitlab.com/SporeDB/sporedb/db"
	"gitlab.com/SporeDB/sporedb/myc/protocol"
)

// Mycelium is the structure used to represent the SporeDB network of nodes.
type Mycelium struct {
	Peers []*Peer
	DB    *db.DB

	transport      transport
	mutex          sync.RWMutex
	rContainer     requestsContainer
	recoveries     map[string]*recovery
	recoveryQuorum int
	fullSyncPeer   string // identity of full sync peer, empty if no full sync required
}

// MyceliumConfig is the structure used to setup a new Mycelium.
type MyceliumConfig struct {
	Listen         string          // Peer API of this mycelium, might be empty to disable listenning.
	Peers          []protocol.Node // Bootstrapping nodes. A connection will be attempted for each node of this slice.
	DB             *db.DB          // Related local database
	RecoveryQuorum int
}

// NewMycelium setups a new Mycelium from its configuration.
func NewMycelium(c *MyceliumConfig) (*Mycelium, error) {
	// Allocate caches
	rc, _ := lru.New(128)

	// Build Mycelium
	m := &Mycelium{
		DB:             c.DB,
		transport:      &transportTCP{},
		rContainer:     requestsContainer{cache: rc},
		recoveryQuorum: c.RecoveryQuorum,
		recoveries:     make(map[string]*recovery),
	}

	if m.recoveryQuorum <= 0 {
		m.recoveryQuorum = 2
	}

	if c.Listen != "" {
		go func() {
			zap.L().Info("Listening",
				zap.String("type", "P2P"),
				zap.String("address", c.Listen),
			)
			lerr := m.transport.Listen(c.Listen, m.handler)
			if lerr != nil {
				zap.L().Error("Unable to listen",
					zap.String("type", "P2P"),
					zap.Error(lerr),
				)
			}
		}()
	}

	// Connect to peers asynchronously
	for _, n := range c.Peers {
		go func(n protocol.Node) {
			_ = m.Bind(&Peer{Node: n})
		}(n)
	}

	// Start broadcaster asynchronously
	go m.broadcaster()

	return m, nil
}

// Bind binds the Mycelium to a new peer.
// It starts a listening routine for the node incoming messages.
func (m *Mycelium) Bind(n *Peer) error {
	// Is already bound?
	m.mutex.Lock()
	for _, p := range m.Peers {
		if n.Equals(p) {
			return nil
		}
	}
	m.mutex.Unlock()

	n.session = protocol.NewECDHESession(m.DB.KeyRing, m.DB.Identity)
	conn, err := m.transport.Bind(n.Address)
	if err != nil {
		return err
	}

	handshake := func() error {
		hello, _ := n.session.Hello()
		c := &protocol.Call{
			F: protocol.FnHELLO,
			M: hello,
		}

		rawTransport := conn.Raw()

		d, _ := c.Pack()
		written, _ := rawTransport.Write(d)
		if written == 0 {
			return errors.New("no connection established")
		}

		err = c.Unpack(rawTransport)
		h, ok := c.M.(*protocol.Hello)
		if err != nil {
			return err
		}

		if !ok {
			return errors.New("invalid hello message")
		}

		err = n.session.Verify(h)
		if err != nil {
			return err
		}

		n.Identity = h.Identity
		zap.L().Info("Handshake",
			zap.String("mode", "client"),
			zap.String("address", n.Address),
			zap.String("identity", n.Identity),
			zap.Bool("trusted", n.session.IsTrusted()),
		)

		return n.session.Open(conn)
	}

	for handshake() != nil {
		time.Sleep(2 * time.Second)
	}

	conn.SetHandshake(handshake)

	n.write = make(chan []byte, 64)

	m.mutex.Lock()
	m.Peers = append(m.Peers, n)
	m.mutex.Unlock()

	go m.router(n)
	return nil
}

// Close shuts down the whole local Mycelium, closing every open connection.
func (m *Mycelium) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, p := range m.Peers {
		_ = p.Close()
	}

	return m.transport.Close()
}

// handler is called for each new incoming connection,
// and each client disconnection (incoming == nil).
// It starts a new listening routine.
func (m *Mycelium) handler(n *Peer, incoming conn) {
	if incoming == nil {
		m.handlerDisconnect(n)
		return
	}

	// Wait for HELLO message (server-side)
	c := &protocol.Call{}
	if c.Unpack(incoming) != nil || c.F != protocol.FnHELLO {
		_ = incoming.Close()
		return
	}

	h, ok := c.M.(*protocol.Hello)
	if !ok {
		_ = incoming.Close()
		return
	}

	n.session = protocol.NewECDHESession(m.DB.KeyRing, m.DB.Identity)
	if n.session.Verify(h) != nil {
		_ = incoming.Close()
		return
	}

	n.Identity = h.Identity

	c.M, _ = n.session.Hello()
	d, _ := c.Pack()
	_, err := incoming.Write(d)
	if err != nil {
		_ = incoming.Close()
		return
	}

	// Is already bound?
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if n.session.Open(incoming) != nil {
		_ = incoming.Close()
		return
	}

	for _, p := range m.Peers {
		if n.Equals(p) {
			return
		}
	}

	n.write = make(chan []byte, 64)
	m.Peers = append(m.Peers, n)
	go m.router(n)
	zap.L().Info("Handshake",
		zap.String("mode", "server"),
		zap.String("address", n.Address),
		zap.String("identity", n.Identity),
		zap.Bool("trusted", n.session.IsTrusted()),
	)
}

func (m *Mycelium) handlerDisconnect(n *Peer) {
	m.mutex.Lock()
	for i, p := range m.Peers {
		if p == n {
			m.Peers = append(m.Peers[:i], m.Peers[i+1:]...)
			break
		}
	}
	m.mutex.Unlock()
	_ = n.Close()
}
