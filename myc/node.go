package myc

// Node is the structure used to represent a node of the Mycelium.
type Node struct {
	Address  string
	Identity string

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
