package myc

import (
	"fmt"
	"sync"
	"time"

	"gitlab.com/SporeDB/sporedb/db"
	"gitlab.com/SporeDB/sporedb/myc/protocol"
)

// Mycelium is the structure used to represent the SporeDB network of nodes.
type Mycelium struct {
	Peers []*Node
	DB    *db.DB

	transport      transport
	recoveries     map[string]*recovery
	mutex          sync.Mutex
	recoveryQuorum int
}

// MyceliumConfig is the structure used to setup a new Mycelium.
type MyceliumConfig struct {
	Listen         string  // Peer API of this mycelium, might be empty to disable listenning.
	Peers          []*Node // Bootstrapping nodes. A connection will be attempted for each node of this slice.
	DB             *db.DB  // Related local database
	RecoveryQuorum int
}

// NewMycelium setups a new Mycelium from its configuration.
func NewMycelium(c *MyceliumConfig) (*Mycelium, error) {
	m := &Mycelium{
		DB:             c.DB,
		transport:      &transportTCP{},
		recoveryQuorum: c.RecoveryQuorum,
		recoveries:     make(map[string]*recovery),
	}
	if m.recoveryQuorum <= 0 {
		m.recoveryQuorum = 2
	}

	if c.Listen != "" {
		go func() { _ = m.transport.Listen(c.Listen, m.handler) }()
	}

	// Connect to peers asynchronously
	for _, n := range c.Peers {
		go func(n *Node) {
			_ = m.Bind(n)
		}(n)
	}

	// Start broadcaster asynchronously
	go m.broadcaster()

	return m, nil
}

// Bind binds the Mycelium to a new node.
// It starts a listening routine for the node incoming messages.
func (m *Mycelium) Bind(n *Node) error {
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
	go m.listener(n)

	fmt.Printf("Bound to %s (%s) in client mode\n", n.Address, n.Identity)
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
func (m *Mycelium) handler(n *Node) {
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
	go m.listener(n)
	fmt.Printf("Bound to %s (%s) in server mode\n", n.Address, n.Identity)
}

func (m *Mycelium) listener(n *Node) {
	go n.emitter()
	for !n.stopped {
		c := &protocol.Call{}
		err := c.Unpack(n.conn)
		if err != nil {
			continue
		}

		switch c.F {
		case protocol.FnSPORE:
			_ = m.DB.Endorse(c.M.(*db.Spore))
		case protocol.FnENDORSE:
			m.DB.AddEndorsement(c.M.(*db.Endorsement))
		case protocol.FnRECOVERREQUEST:
			m.handleRecoverRequest(n, c.M.(*db.RecoverRequest))
		case protocol.FnRAW:
			m.handleRaw(n.Identity, c.M.(*protocol.Raw))
		}
	}
}

func (m *Mycelium) broadcaster() {
	for message := range m.DB.Messages {
		c := &protocol.Call{M: message}
		if _, ok := message.(*db.Spore); ok {
			c.F = protocol.FnSPORE
		} else if _, ok := message.(*db.Endorsement); ok {
			c.F = protocol.FnENDORSE
		} else if r, ok := message.(*db.RecoverRequest); ok {
			c.F = protocol.FnRECOVERREQUEST
			m.StartRecovery(r.Key, m.recoveryQuorum)
		} else {
			continue
		}

		data, err := c.Pack()
		if err != nil {
			continue
		}

		m.mutex.Lock()
		for len(m.Peers) == 0 { // No peer available, wait for broadcast retry
			m.mutex.Unlock()
			time.Sleep(5 * time.Second)
			m.mutex.Lock()
		}

		for _, p := range m.Peers {
			p.write <- data
		}
		m.mutex.Unlock()
	}
}
