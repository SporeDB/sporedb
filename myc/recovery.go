package myc

import (
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

func (m *Mycelium) handleRecoverRequest(node *Node, request *db.RecoverRequest) {
	data, v, err := m.DB.Get(request.Key)
	if err != nil {
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

	rawMessage, err := (&protocol.Call{
		F: protocol.FnRAW,
		M: raw,
	}).Pack()

	if err != nil {
		return
	}

	node.write <- rawMessage
}

func (m *Mycelium) handleRaw(identity string, raw *protocol.Raw) {
	if identity == "" {
		return
	}

	m.mutex.Lock()
	r, ok := m.recoveries[raw.Key]
	m.mutex.Unlock()
	if !ok {
		return
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify recovery timeout
	if r.stale || r.deadline.Before(time.Now()) {
		r.stale = true
		m.StopRecovery(raw.Key)
		return
	}

	// Verify version
	if version.New(raw.Data).Matches(raw.Version) != nil {
		zap.L().Warn("Invalid recovery proposal",
			zap.String("emitter", identity),
			zap.String("step", "version"),
		)
		return
	}

	// Verify raw signature
	err := m.DB.KeyRing.Verify(identity, raw.GetMessage(), raw.Signature)
	if err != nil {
		zap.L().Warn("Invalid recovery proposal",
			zap.String("emitter", identity),
			zap.String("step", "crypto"),
			zap.Error(err),
		)
		return
	}

	r.answers[identity] = raw

	if len(r.answers) >= r.quorum {
		result := r.checkQuorum()
		if result != nil {
			_ = m.DB.Apply(&db.Spore{
				Uuid: "RECOVER_" + result.Key,
				Operations: []*db.Operation{{
					Key:  result.Key,
					Op:   db.Operation_SET,
					Data: result.Data,
				}},
			})
			m.StopRecovery(raw.Key)
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

	zap.L().Info("Aborting partial recovery",
		zap.String("key", key),
		zap.Int("quorum", r.quorum),
		zap.Int("answers", len(r.answers)),
		zap.Bool("aborted", r.deadline.Before(time.Now())),
	)
	delete(m.recoveries, key)
}
