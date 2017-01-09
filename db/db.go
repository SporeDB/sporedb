package db

import (
	"regexp"
	"sync"
	"time"
)

// DB is the main structure for database management of a node.
type DB struct {
	Store Store

	// Policy management
	policies    map[string]*Policy
	policiesReg map[string][]*regexp.Regexp

	// Spore flow management
	staging      map[string]*dbTrigger
	stagingMutex sync.RWMutex
	waiting      map[string]*dbTrigger
	waitingMutex sync.RWMutex
	gc           chan *Spore
}

type dbTrigger struct {
	timer   *time.Timer
	channel chan *Endorsement
	spore   *Spore
}

// NewDB instanciates a new database with clean initialization.
func NewDB(s Store) *DB {
	return &DB{
		Store:       s,
		policies:    make(map[string]*Policy),
		policiesReg: make(map[string][]*regexp.Regexp),
		staging:     make(map[string]*dbTrigger),
		waiting:     make(map[string]*dbTrigger),
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
	wg.Add(1)

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
					v.channel <- db.executeEndorsement(v.spore)
					delete(db.waiting, k)
				} else if err == ErrDeadlineExpired {
					v.channel <- nil
					delete(db.waiting, k)
				} else {
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
}
