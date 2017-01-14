package myc

import (
	"fmt"
	"sync"

	"gitlab.com/SporeDB/sporedb/db"
	"gitlab.com/SporeDB/sporedb/myc/protocol"
)

// Mycelium is the structure used to represent the SporeDB network of nodes.
type Mycelium struct {
	Peers []*Node
	DB    *db.DB

	transport transport
	mutex     sync.Mutex
}

// MyceliumConfig is the structure used to setup a new Mycelium.
type MyceliumConfig struct {
	Listen string  // Peer API of this mycelium, might be empty to disable listenning.
	Peers  []*Node // Bootstrapping nodes. A connection will be attempted for each node of this slice.
	DB     *db.DB  // Related local database
}

// NewMycelium setups a new Mycelium from its configuration.
func NewMycelium(c *MyceliumConfig) (*Mycelium, error) {
	m := &Mycelium{DB: c.DB, transport: &transportTCP{}}

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

	handshake := func() error {
		// Try to perform HELLO exchange.
		c := &protocol.Call{F: protocol.FnHELLO, M: &protocol.Hello{Version: protocol.Version}}
		d, _ := c.Pack()
		_, err = conn.Write(d)
		if err != nil {
			_ = conn.Close()
			return err
		}

		err = c.Unpack(conn)
		h, ok := c.M.(*protocol.Hello)
		if err != nil || !ok || h.Version != protocol.Version {
			_ = conn.Close()
			return err
		}
		return nil
	}

	err = handshake()
	if err != nil {
		_ = conn.Close()
		return err
	}

	conn.SetHandshake(handshake)
	n.conn = conn
	n.write = make(chan []byte, 64)
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.Peers = append(m.Peers, n)
	go m.listener(n)

	fmt.Println("Bound to", n.Address)
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
	// Wait for HELLO message
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

	// Echo
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
	fmt.Println("Bound to", n.Address)
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
		} else {
			continue
		}

		data, err := c.Pack()
		if err != nil {
			continue
		}

		m.mutex.Lock()
		for _, p := range m.Peers {
			p.write <- data
		}
		m.mutex.Unlock()
	}
}

// Node is the structure used to represent a node of the Mycelium.
type Node struct {
	Address string

	write   chan []byte
	conn    conn
	stopped bool
}

func (n *Node) emitter() {
	for data := range n.write {
		_, _ = n.conn.Write(data)
	}
}

// Close properly shuts down node's connection.
func (n *Node) Close() error {
	close(n.write)
	n.stopped = true
	return n.conn.Close()
}

// Equals is used to differentiate two nodes.
func (n *Node) Equals(n2 *Node) bool {
	return n.Address == n2.Address
}
