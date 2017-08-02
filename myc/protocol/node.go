package protocol

// Equals shall be used to compare two nodes.
func (n Node) Equals(n2 Node) bool {
	return n.Address == n2.Address
}
