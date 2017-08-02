package myc

import (
	"go.uber.org/zap"

	"gitlab.com/SporeDB/sporedb/db"
	"gitlab.com/SporeDB/sporedb/myc/protocol"
)

// Broadcast may be used to send a protocol call to connected peers.
// If `from` is not nil, no message will be sent to the provided peer.
func (m *Mycelium) Broadcast(from *Peer, call *protocol.Call) (sent int) {
	data, err := call.Pack()
	if err != nil {
		zap.L().Error("Unable to pack message",
			zap.String("type", call.F.String()),
			zap.String("step", "broadcast"),
			zap.Error(err),
		)
		return
	}

	// Shall we cache the data for further use?
	if call.F == protocol.FnSPORE {
		spore := call.M.(*db.Spore)
		m.rContainer.SetDelivered(spore.Uuid, data)
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, p := range m.Peers {
		if p == from { // Do not re-send to the peer that initially sent the message
			continue
		}

		p.write <- data
		sent++
	}

	return
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

		m.Broadcast(nil, c)
	}
}
