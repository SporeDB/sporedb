// Package myc contains the SporeDB mycelium logic.
package myc

import (
	"errors"
	"math/rand"
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
	Nodes []protocol.Node
	DB    *db.DB

	ticker       *time.Ticker
	random       *rand.Rand
	transport    transport
	mutex        sync.RWMutex
	rContainer   requestsContainer
	recoveries   map[string]*recovery
	fullSyncPeer string // identity of full sync peer, empty if no full sync required
	listenAddr   string

	connectivity   int
	fanout         int
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

	// Copy node configuration
	nodes := make([]protocol.Node, len(c.Peers))
	copy(nodes, c.Peers)

	// Build Mycelium
	m := &Mycelium{
		Nodes:          nodes,
		DB:             c.DB,
		ticker:         time.NewTicker(10 * time.Second),
		random:         rand.New(rand.NewSource(time.Now().UnixNano())),
		transport:      &transportTCP{},
		rContainer:     requestsContainer{cache: rc},
		recoveryQuorum: c.RecoveryQuorum,
		recoveries:     make(map[string]*recovery),
		listenAddr:     c.Listen,
		connectivity:   10,
		fanout:         10,
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

	go m.membershipConnecter()
	go m.membershipBroadcaster()
	go m.broadcaster()

	return m, nil
}

// Bind binds the Mycelium to a specific peer.
// It starts a listening routine for the node incoming messages.
func (m *Mycelium) Bind(n protocol.Node) error {
	p := &Peer{Node: n}

	// Is already bound?
	m.mutex.Lock()
	for _, p2 := range m.Peers {
		if p.Equals(p2) {
			return nil
		}
	}
	m.Peers = append(m.Peers, p)
	m.mutex.Unlock()

	p.session = protocol.NewECDHESession(m.DB.KeyRing, m.DB.Identity)
	conn, err := m.transport.Bind(p.Address)
	if err != nil {
		return err
	}

	handshake := func() error {
		hello, _ := p.session.Hello()
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

		err = p.session.Verify(h)
		if err != nil {
			return err
		}

		p.Identity = h.Identity
		zap.L().Info("Handshake",
			zap.String("mode", "client"),
			zap.String("address", p.Address),
			zap.String("identity", p.Identity),
			zap.Bool("trusted", p.session.IsTrusted()),
		)

		return p.session.Open(conn)
	}

	for handshake() != nil {
		time.Sleep(2 * time.Second)
	}

	conn.SetHandshake(handshake)

	p.write = make(chan []byte, 64)

	m.mutex.Lock()
	p.ready = true
	m.mutex.Unlock()

	go m.router(p)
	return nil
}

// Close shuts down the whole local Mycelium, closing every open connection.
func (m *Mycelium) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, p := range m.Peers {
		_ = p.Close()
	}

	m.ticker.Stop()
	return m.transport.Close()
}

// handler is called for each new incoming connection,
// and each client disconnection (incoming == nil).
// It starts a new listening routine.
func (m *Mycelium) handler(p *Peer, incoming conn) {
	if incoming == nil {
		m.handlerDisconnect(p)
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

	p.session = protocol.NewECDHESession(m.DB.KeyRing, m.DB.Identity)
	if p.session.Verify(h) != nil {
		_ = incoming.Close()
		return
	}

	p.Identity = h.Identity

	c.M, _ = p.session.Hello()
	d, _ := c.Pack()
	_, err := incoming.Write(d)
	if err != nil {
		_ = incoming.Close()
		return
	}

	// Is already bound?
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if p.session.Open(incoming) != nil {
		_ = incoming.Close()
		return
	}

	for _, p2 := range m.Peers {
		if p.Equals(p2) {
			return
		}
	}

	p.ready = true
	p.write = make(chan []byte, 64)
	m.Peers = append(m.Peers, p)

	go m.router(p)
	zap.L().Info("Handshake",
		zap.String("mode", "server"),
		zap.String("address", p.Address),
		zap.String("identity", p.Identity),
		zap.Bool("trusted", p.session.IsTrusted()),
	)
}

func (m *Mycelium) handlerDisconnect(p *Peer) {
	m.mutex.Lock()
	for i, p2 := range m.Peers {
		if p == p2 {
			m.Peers = append(m.Peers[:i], m.Peers[i+1:]...)
			break
		}
	}
	m.mutex.Unlock()
	_ = p.Close()
}
