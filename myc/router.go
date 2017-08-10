package myc

import (
	"time"

	"go.uber.org/zap"

	"gitlab.com/SporeDB/sporedb/db"
	"gitlab.com/SporeDB/sporedb/myc/protocol"
)

func (m *Mycelium) router(p *Peer) {
	go p.emitter()
	for !p.stopped {
		c := &protocol.Call{}
		err := c.Unpack(p.conn)
		if err != nil {
			continue
		}

		zap.L().Debug("P2P",
			zap.String("type", c.F.String()),
			zap.String("address", p.Address),
			zap.String("identity", p.Identity),
		)

		switch c.F {
		case protocol.FnSPORE:
			go m.handleSpore(p, c.M.(*db.Spore))
		case protocol.FnENDORSE:
			go m.handleEndorsement(p, c.M.(*db.Endorsement))
		case protocol.FnGOSSIP:
			g := c.M.(*protocol.Gossip)
			if g.Request {
				go m.handleGossipRequest(p, g)
			} else {
				go m.handleGossipProposal(p, g)
			}
		case protocol.FnRECOVERREQUEST:
			go m.handleRecoverRequest(p, c.M.(*db.RecoverRequest))
		case protocol.FnRAW:
			go m.handleRaw(p.Identity, c.M.(*protocol.Raw))
		case protocol.FnCATALOG:
			go m.handleCatalog(p.Identity, c.M.(*db.Catalog))
		}
	}
}

func (m *Mycelium) handleSpore(p *Peer, s *db.Spore) {
	if nil == m.DB.Endorse(s) {
		call := &protocol.Call{
			F: protocol.FnSPORE,
			M: s,
		}

		data, err := call.Pack()
		if err != nil {
			zap.L().Error("Unable to pack message",
				zap.String("type", call.F.String()),
				zap.String("step", "handleSpore"),
				zap.Error(err),
			)
		} else {
			m.rContainer.SetDelivered(s.Uuid, data)
		}

		m.Broadcast(p, &protocol.Call{
			F: protocol.FnGOSSIP,
			M: &protocol.Gossip{
				Spores: []string{s.Uuid},
			},
		})
	}
}

func (m *Mycelium) handleEndorsement(p *Peer, e *db.Endorsement) {
	for i := 0; i < 20; i++ { // See https://gitlab.com/SporeDB/sporedb/issues/5
		err := m.DB.AddEndorsement(e)
		if err == nil {
			m.Broadcast(p, &protocol.Call{
				F: protocol.FnENDORSE,
				M: e,
			})
			return
		}
		if err != db.ErrNoRelatedSpore {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (m *Mycelium) handleGossipProposal(p *Peer, g *protocol.Gossip) {
	for _, sporeUUID := range g.Spores {
		if ok, _ := m.rContainer.IsDelivered(sporeUUID); ok {
			continue
		}

		if m.rContainer.Add(sporeUUID, p.Node) {
			g.Request = true
			call := &protocol.Call{
				F: protocol.FnGOSSIP,
				M: g,
			}

			data, err := call.Pack()
			if err != nil {
				zap.L().Error("Unable to pack message",
					zap.String("type", call.F.String()),
					zap.String("step", "handleSpore"),
					zap.Error(err),
				)
				return
			}

			p.write <- data
		}
	}
}

func (m *Mycelium) handleGossipRequest(p *Peer, g *protocol.Gossip) {
	for _, sporeUUID := range g.Spores {
		ok, data := m.rContainer.IsDelivered(sporeUUID)
		if !ok {
			zap.L().Warn("Gossip miss",
				zap.String("uuid", sporeUUID),
			)
			continue
		}

		p.write <- data
	}
}
