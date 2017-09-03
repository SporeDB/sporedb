package myc

import (
	"time"

	"gitlab.com/SporeDB/sporedb/myc/protocol"
	"go.uber.org/zap"
)

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

				var connected bool
				for _, p := range m.Peers {
					if p.Node.Equals(n) {
						connected = true
						break
					}
				}

				if !connected {
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
