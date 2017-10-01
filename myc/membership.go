package myc

import (
	"time"

	"gitlab.com/SporeDB/sporedb/myc/protocol"
	"go.uber.org/zap"
)

func peerConnected(peers []*Peer, node protocol.Node) (connected bool) {
	for _, p := range peers {
		if p.Node.Equals(node) {
			connected = true
			break
		}
	}
	return
}

// membershipConnecter ensures that the current node is bound to enough peers.
// The list of available nodes shall be updated by connected peers using broadcast.
func (m *Mycelium) membershipConnecter() {
	for {
		var nodeToBind protocol.Node

		m.mutex.Lock()
		if len(m.Peers) < m.connectivity {
			permutations := m.random.Perm(len(m.Nodes))
			for _, i := range permutations {
				n := m.Nodes[i]

				if !peerConnected(m.Peers, n) {
					nodeToBind = n
					break
				}
			}
		}
		m.mutex.Unlock()

		if !nodeToBind.Zero() {
			err := m.Bind(nodeToBind)
			if err != nil {
				zap.L().Warn("Binding error",
					zap.String("address", nodeToBind.Address),
					zap.String("identity", nodeToBind.Identity),
					zap.Error(err))
			}
			time.Sleep(time.Second)
		} else { // no need to bind a new node or no node available
			time.Sleep(10 * time.Second)
		}
	}
}

func (m *Mycelium) membershipBroadcaster() {
	for range m.ticker.C {
		m.mutex.RLock()

		var nodes []*protocol.Node
		permutations := m.random.Perm(len(m.Nodes))
		for i, j := range permutations {
			if i >= m.fanout {
				break
			}

			nodes = append(nodes, &m.Nodes[j])
		}

		if len(nodes) > 0 {
			m.Broadcast(nil, &protocol.Call{
				F: protocol.FnNODES,
				M: &protocol.Nodes{Nodes: nodes},
			})
		}

		m.mutex.RUnlock()
	}
}

func (m *Mycelium) filterIncomingNodes(nodes *protocol.Nodes) []*protocol.Node {
	var notConnected []*protocol.Node
	for _, node := range nodes.Nodes {
		if node.Address != "" && !peerConnected(m.Peers, *node) && !m.selfAddresses[node.Address] {
			notConnected = append(notConnected, node)
		}
	}
	return notConnected
}

func (m *Mycelium) handleNodes(p *Peer, nodes *protocol.Nodes) {
	if !p.session.IsTrusted() {
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Filter out connected peers and bootstraping nodes
	var notConnected = m.filterIncomingNodes(nodes)

	// Update nodes
	for _, node := range notConnected {
		m.refreshNodes(*node)
	}
}

// refreshNodes refresh mycelium cold nodes list.
// it must be executed within a mutex.
func (m *Mycelium) refreshNodes(node protocol.Node) {
	var found bool
	for i, storedNode := range m.Nodes {
		if (node.Identity != "" && node.Identity == storedNode.Identity) ||
			(node.Address == storedNode.Address) {
			zap.L().Info("Updating node information",
				zap.String("identity", node.Identity),
				zap.String("address", node.Address),
			)
			m.Nodes[i] = node
			found = true
		}
	}

	if !found {
		zap.L().Info("Adding new node information",
			zap.String("identity", node.Identity),
			zap.String("address", node.Address),
		)
		m.Nodes = append(m.Nodes, node)
	}
}
