// Package myc contains the SporeDB mycelium logic.
package myc

import (
	"sync"

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

	conn, err := m.transport.Bind(n.Address)
	if err != nil {
		return err
	}

	handshake := func() (error, string) {
		// Try to perform HELLO exchange from client-side.
		// During the handshake, remote version and identity are fetched.
		c := &protocol.Call{
			F: protocol.FnHELLO,
			M: &protocol.Hello{
				Version:  protocol.Version,
				Identity: m.DB.Identity,
			},
		}

		d, _ := c.Pack()
		_, err = conn.Write(d)
		if err != nil {
			_ = conn.Close()
			return err, ""
		}

		err = c.Unpack(conn)
		h, ok := c.M.(*protocol.Hello)
		if err != nil || !ok || h.Version != protocol.Version {
			_ = conn.Close()
			return err, ""
		}
		return nil, h.Identity
	}

	err, identity := handshake()
	if err != nil {
		_ = conn.Close()
		return err
	}

	conn.SetHandshake(func() error {
		err, _ := handshake()
		return err
	})

	n.Identity = identity
	n.conn = conn
	n.write = make(chan []byte, 64)

	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.Peers = append(m.Peers, n)
	go m.router(n)

	zap.L().Info("Bound",
		zap.String("mode", "client"),
		zap.String("address", n.Address),
		zap.String("identity", n.Identity),
	)
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

// handler is called for each new incoming connection.
// It starts a new listening routine.
func (m *Mycelium) handler(n *Peer) {
	// Wait for HELLO message (server-side)
	c := &protocol.Call{}
	err := c.Unpack(n.conn)
	if err != nil || c.F != protocol.FnHELLO {
		_ = n.conn.Close()
		return
	}

	h, ok := c.M.(*protocol.Hello)
	if !ok || h.Version != protocol.Version {
		_ = n.conn.Close()
		return
	}
	n.Identity = h.Identity

	// Update identity and reply
	h.Identity = m.DB.Identity
	d, _ := c.Pack()
	_, err = n.conn.Write(d)
	if err != nil {
		_ = n.conn.Close()
		return
	}

	// Is already bound?
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, p := range m.Peers {
		if n.Equals(p) {
			return
		}
	}

	n.write = make(chan []byte, 64)
	m.Peers = append(m.Peers, n)
	go m.router(n)
	zap.L().Info("Bound",
		zap.String("mode", "server"),
		zap.String("address", n.Address),
		zap.String("identity", n.Identity),
	)
}
