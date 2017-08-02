package myc

import "gitlab.com/SporeDB/sporedb/myc/protocol"

// Peer is the structure used to represent a peer of the Mycelium.
type Peer struct {
	protocol.Node

	write   chan []byte
	conn    conn
	stopped bool
}

func (p *Peer) emitter() {
	for data := range p.write {
		_, _ = p.conn.Write(data)
	}
}

// Close properly shuts down node's connection.
func (p *Peer) Close() error {
	close(p.write)
	p.stopped = true
	return p.conn.Close()
}

// Equals shall be used to compare two peers.
func (p *Peer) Equals(p2 *Peer) bool {
	return p.Node.Equals(p2.Node)
}
