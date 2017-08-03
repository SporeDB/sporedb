// Package db provides main consensus and data processing engines.
package db

import (
	"crypto/sha512"
	"regexp"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/golang/protobuf/proto"
	lru "github.com/hashicorp/golang-lru"

	"gitlab.com/SporeDB/sporedb/db/operations"
	"gitlab.com/SporeDB/sporedb/db/version"
	"gitlab.com/SporeDB/sporedb/myc/sec"
)

// DB is the main structure for database management of a node.
type DB struct {
	// Store is the underlying database storage engine.
	Store Store

	// Identity is the identity of the local node.
	// It should be unique.
	Identity string

	// KeyRing is the key management structure, used to
	// sign and verify endorsements and spores.
	KeyRing sec.KeyRing

	// Messages is the output port of the consensus algorithm.
	// It emits various messages, like new Spores or Endorsements.
	//
	// See gitlab.com/SporeDB/sporedb/myc/protocol
	Messages chan proto.Message

	// Policy management
	policies    map[string]*Policy
	policiesReg map[string][]*regexp.Regexp

	// Spore flow management
	//
	// Those 3 maps are the basic blocks of the SporeDB consensus algorithm.
	//
	// * `waiting` contains spores that are conflicting with one or more spores
	//   in `staging`. They are either dropped or promoted to `staging`, depending
	//   on the fate of their conflicting peers ;
	//
	// * `staging` contains spores that have been validated, but requires more
	//    endorsements to be trusted. They are either dropped or promoted to
	//    `applied`.
	//
	// * `applied` contains grace-period information about Spores that have been applied
	//   recently. When the grace-period is over, Spores are removed to free-up some
	//   space.
	//
	waiting map[string]*dbTrigger
	staging map[string]*dbTrigger
	applied map[string]time.Time

	// Paralellism management
	waitingMutex sync.RWMutex
	stagingMutex sync.RWMutex
	appliedMutex sync.Mutex
	cache        *lru.Cache
	gc           chan *Spore
	cleanTicker  *time.Ticker
}

type dbTrigger struct {
	timer        *time.Timer
	endorsements []*Endorsement
	spore        *Spore
}

// NewDB instanciates a new database with clean initialization.
func NewDB(s Store, identity string, keyring sec.KeyRing) *DB {
	c, _ := lru.New(32)
	return &DB{
		Store:       s,
		Identity:    identity,
		KeyRing:     keyring,
		Messages:    make(chan proto.Message, 16),
		policies:    make(map[string]*Policy),
		policiesReg: make(map[string][]*regexp.Regexp),
		staging:     make(map[string]*dbTrigger),
		waiting:     make(map[string]*dbTrigger),
		applied:     make(map[string]time.Time),
		cache:       c,
		gc:          make(chan *Spore),
	}
}

// AddPolicy registers a new policy for the database.
func (db *DB) AddPolicy(p *Policy) error {
	regexes, err := p.compileRegexes()
	if err != nil {
		return err
	}

	db.policies[p.Uuid], db.policiesReg[p.Uuid] = p, regexes
	return nil
}

// Start starts the database, waiting for incoming spores to be processed.
// It can either work in blocking or non-blocking modes.
func (db *DB) Start(blocking bool) {
	var wg sync.WaitGroup
	wg.Add(2)

	db.cleanTicker = time.NewTicker(60 * time.Second)
	go func() {
		for range db.cleanTicker.C {
			db.Clean()
		}
		wg.Done()
	}()

	go func() {
		for s := range db.gc {
			// Delete expired Spore
			db.stagingMutex.Lock()
			delete(db.staging, s.Uuid)
			db.stagingMutex.Unlock()

			// Lock the whole block for access to waiting list
			db.waitingMutex.Lock()

			for k, v := range db.waiting {
				err := db.CanEndorse(v.spore)
				if err == nil {
					db.executeEndorsement(v.spore)
					delete(db.waiting, k)
				} else if err == ErrDeadlineExpired {
					delete(db.waiting, k)
				}
			}

			db.waitingMutex.Unlock()
		}
		wg.Done()
	}()

	if blocking {
		wg.Wait()
	}
}

// Stop asks the database to be gracefully stopped.
func (db *DB) Stop() {
	close(db.gc)
	db.cleanTicker.Stop()
}

// Get returns the currently stored data for the provided key.
func (db *DB) Get(key string) ([]byte, *version.V, error) {
	return db.Store.Get(key)
}

// Apply directly applies the Spore's operations to the database (atomic).
func (db *DB) Apply(s *Spore) error {
	db.Store.Lock()
	defer db.Store.Unlock()

	db.appliedMutex.Lock()
	defer db.appliedMutex.Unlock()

	policy := db.policies[s.Policy]
	ok, unixTime := s.checkGracePeriod(policy.GracePeriod)
	if !ok {
		zap.L().Warn("Grace period expired",
			zap.String("uuid", s.Uuid),
			zap.Time("death", unixTime),
		)
		return ErrGracePeriodExpired
	}

	if _, ok := db.applied[s.Uuid]; ok {
		zap.L().Warn("Double application attempt",
			zap.String("uuid", s.Uuid),
		)
		return ErrDuplicatedApplication
	}

	values := make(map[string]*operations.Value)

	for _, op := range s.Operations {
		value, ok := values[op.Key]
		if !ok {
			data, v, err := db.Store.Get(op.Key)
			if err != nil && v != version.NoVersion {
				return err
			}

			values[op.Key] = operations.NewValue(data)
			value = values[op.Key]
		}

		err := op.Exec(value)
		if err != nil {
			return err
		}
	}

	keys := make([]string, len(values))
	rawValues := make([][]byte, len(values))
	versions := make([]*version.V, len(values))

	var i int
	for k, v := range values {
		keys[i] = k
		rawValues[i] = v.Raw
		versions[i] = version.New(v.Raw)
		i++
	}

	zap.L().Info("Apply",
		zap.String("uuid", s.Uuid),
	)
	err := db.Store.SetBatch(keys, rawValues, versions)
	if err != nil {
		zap.L().Error("Application error",
			zap.String("uuid", s.Uuid),
			zap.Error(err),
		)
		return err
	}

	db.applied[s.Uuid] = unixTime
	return nil
}

// Clean is periodically called to free-up memory related to old transactions.
func (db *DB) Clean() {
	db.appliedMutex.Lock()
	defer db.appliedMutex.Unlock()

	unixZero := time.Unix(0, 0)
	now := time.Now()

	for uuid, death := range db.applied {
		if death.After(unixZero) && death.After(now) {
			delete(db.applied, uuid)
		}
	}
}

// HashSpore process one spore's hash.
// It stores and caches the value for efficient computation.
func (db *DB) HashSpore(s *Spore) []byte {
	hash, ok := db.cache.Get(s.Uuid)
	if ok {
		return hash.([]byte)
	}

	newHash := hashMessage(s)
	go db.cache.Add(s.Uuid, newHash)
	return newHash
}

func hashMessage(message proto.Message) []byte {
	raw, _ := proto.Marshal(message)
	hash := sha512.Sum512(raw)
	return hash[:]
}
