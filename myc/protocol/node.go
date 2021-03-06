package protocol

// Equals shall be used to compare two nodes.
func (n Node) Equals(n2 Node) bool {
	return n.Address != "" && n.Address == n2.Address ||
		n.Identity != "" && n.Identity == n2.Identity
}

// Zero returns true if n is the zero value for nodes.
func (n Node) Zero() bool {
	return len(n.Address) == 0
}
