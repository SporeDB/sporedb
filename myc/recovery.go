package myc

import (
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"gitlab.com/SporeDB/sporedb/db"
	"gitlab.com/SporeDB/sporedb/db/version"
	"gitlab.com/SporeDB/sporedb/myc/protocol"
)

type recovery struct {
	deadline time.Time
	answers  map[string]*protocol.Raw
	mutex    sync.Mutex
	quorum   int
	stale    bool
}

func (m *Mycelium) handleRecoverRequest(peer *Peer, request *db.RecoverRequest) {
	var err error
	var call *protocol.Call

	if request.Key == "" { // Full-State-Transfer request, send catalog
		catalog := &db.Catalog{}
		catalog.Keys, err = m.DB.Store.List()
		if err != nil {
			zap.L().Error("Unable to send the catalog",
				zap.Error(err),
			)
			return
		}

		zap.L().Info("Sending catalog",
			zap.String("peer", peer.Identity),
		)

		call = &protocol.Call{
			F: protocol.FnCATALOG,
			M: catalog,
		}
	} else {
		var data []byte
		var v *version.V
		data, v, err = m.DB.Get(request.Key)
		if err != nil {
			zap.L().Error("Unable to get key for recovery",
				zap.String("key", request.Key),
				zap.Error(err),
			)
			return
		}

		raw := &protocol.Raw{
			Key:     request.Key,
			Version: v,
			Data:    data,
		}

		raw.Signature, err = m.DB.KeyRing.Sign(raw.GetMessage())
		if err != nil {
			zap.L().Error("Unable to sign the spore",
				zap.String("step", "recovery_proposal"),
				zap.Error(err),
			)
			return
		}

		call = &protocol.Call{
			F: protocol.FnRAW,
			M: raw,
		}
	}

	rawMessage, err := call.Pack()
	if err != nil {
		zap.L().Error("Unable to pack recovery response",
			zap.String("type", call.F.String()),
			zap.Error(err),
		)
		return
	}

	peer.write <- rawMessage
}

func (m *Mycelium) handleRaw(peer *Peer, raw *protocol.Raw) {
	if !peer.session.IsTrusted() {
		return
	}

	m.mutex.Lock()
	r, ok := m.recoveries[raw.Key]
	m.mutex.Unlock()
	if !ok {
		return
	}

	r.mutex.Lock() // Lock the recovery
	defer r.mutex.Unlock()

	// Verify recovery timeout
	if r.stale || r.deadline.Before(time.Now()) {
		r.stale = true
		zap.L().Warn("Recovery expired",
			zap.String("key", raw.Key),
			zap.Time("deadline", r.deadline),
		)
		m.StopRecovery(raw.Key)
		return
	}

	// Verify version
	if !strings.HasPrefix(raw.Key, db.InternalKeyPrefix) && version.New(raw.Data).Matches(raw.Version) != nil {
		zap.L().Warn("Invalid recovery proposal",
			zap.String("key", raw.Key),
			zap.String("emitter", peer.Identity),
			zap.String("step", "version"),
		)
		return
	}

	// Verify raw signature
	err := m.DB.KeyRing.Verify(peer.Identity, raw.GetMessage(), raw.Signature)
	if err != nil {
		zap.L().Warn("Invalid recovery proposal",
			zap.String("key", raw.Key),
			zap.String("emitter", peer.Identity),
			zap.String("step", "crypto"),
			zap.Error(err),
		)
		return
	}

	r.answers[peer.Identity] = raw

	if len(r.answers) >= r.quorum {
		result := r.checkQuorum()
		if result != nil {
			_ = m.DB.Store.Set(result.Key, result.Data, result.Version)
			m.StopRecovery(raw.Key)
		}
	}
}

func (m *Mycelium) handleCatalog(peer *Peer, catalog *db.Catalog) {
	m.mutex.Lock()
	fullSyncPeer := m.fullSyncPeer

	if peer.Identity != fullSyncPeer || !peer.session.IsTrusted() {
		m.mutex.Unlock()
		return
	}

	m.fullSyncPeer = ""
	m.mutex.Unlock()

	for k, v := range catalog.Keys {
		_, v2, _ := m.DB.Store.Get(k)
		if v.Matches(v2) != nil {
			m.StartRecovery(k, m.recoveryQuorum)
			m.Broadcast(nil, &protocol.Call{
				F: protocol.FnRECOVERREQUEST,
				M: &db.RecoverRequest{
					Key: k,
				},
			})
		}
	}
}

func (r *recovery) checkQuorum() *protocol.Raw {
	count := make(map[string]int)
	raws := make(map[string]*protocol.Raw)
	for _, r := range r.answers {
		id := r.Version.String()
		count[id]++
		raws[id] = r
	}

	for id, c := range count {
		if c >= r.quorum {
			return raws[id]
		}
	}

	return nil
}

// StartFullSync starts a full state-transfer recovery by asking a (hopefully)
// trusted node his full catalog of (key, version) pairs.
func (m *Mycelium) StartFullSync(peer string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Looking for the peer
	var peerChan chan []byte
	var peerTrusted bool
	for _, p := range m.Peers {
		if p.Identity == peer {
			peerChan = p.write
			peerTrusted = p.session.IsTrusted()
			break
		}
	}

	if peerChan == nil {
		zap.L().Warn("Unable to find full state-transfer peer",
			zap.String("peer", peer),
		)
		return
	}

	if !peerTrusted {
		zap.L().Warn("Untrusted full state-transfer peer",
			zap.String("peer", peer),
		)
		return
	}

	call := &protocol.Call{
		F: protocol.FnRECOVERREQUEST,
		M: &db.RecoverRequest{},
	}
	data, _ := call.Pack()
	peerChan <- data
	m.fullSyncPeer = peer

	zap.L().Info("Asking for a full state-transfer",
		zap.String("peer", peer),
	)
}

// StartRecovery registers a new recovery process for the specified key.
// The new value will be considered trusted if at least "quorum" answers are identical.
func (m *Mycelium) StartRecovery(key string, quorum int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	_, ok := m.recoveries[key]
	if ok {
		return
	}

	m.recoveries[key] = &recovery{
		deadline: time.Now().Add(time.Minute),
		answers:  make(map[string]*protocol.Raw),
		quorum:   quorum,
	}

	zap.L().Info("Starting partial recovery",
		zap.String("key", key),
		zap.Int("quorum", quorum),
		zap.Time("deadline", m.recoveries[key].deadline),
	)
}

// StopRecovery aborts a recovery process for the specified key.
func (m *Mycelium) StopRecovery(key string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	r, ok := m.recoveries[key]
	if !ok {
		return
	}

	zap.L().Info("Stopping partial recovery",
		zap.String("key", key),
		zap.Int("quorum", r.quorum),
		zap.Int("answers", len(r.answers)),
		zap.Bool("aborted", r.deadline.Before(time.Now())),
	)
	delete(m.recoveries, key)
}
