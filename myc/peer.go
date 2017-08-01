package myc

import "gitlab.com/SporeDB/sporedb/myc/protocol"

// Peer is the structure used to represent a peer of the Mycelium.
type Peer struct {
	protocol.Node

	write   chan []byte
	conn    conn
	stopped bool
}

func (n *Peer) emitter() {
	for data := range n.write {
		_, _ = n.conn.Write(data)
	}
}

// Close properly shuts down node's connection.
func (n *Peer) Close() error {
	close(n.write)
	n.stopped = true
	return n.conn.Close()
}

// Equals is used to differentiate two nodes.
func (n *Peer) Equals(n2 *Peer) bool {
	return n.Node.Address == n2.Node.Address
}
