package db

import (
	"errors"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
)

// ErrDeadlineExpired is returned when a spore cannot be endorsed due to an expired deadline according to local clock.
var ErrDeadlineExpired = errors.New("unable to endorse a spore with expired deadline")

// ErrConflictingWithStaging is returned when a node cannot endorse a spore because it already has endorsed a conflicting spore.
var ErrConflictingWithStaging = errors.New("unable to endorse a spore due to conflicting promise")

// CanEndorse checks wether a Spore can be endorsed or not regarding current database status.
// It is thread-safe.
func (db *DB) CanEndorse(s *Spore) error {
	// Timeout: Check deadline
	if !s.checkDeadline() {
		return ErrDeadlineExpired
	}

	// Consistency: Check that the operations are no behind the state and fulfill the types
	db.Store.Lock()
	for k, v := range s.Requirements {
		_, v2, err := db.Store.Get(k)
		if err != nil {
			db.Store.Unlock()
			return err
		}

		if v2.Matches(v) != nil {
			db.Store.Unlock()
			return err
		}
	}

	for _, op := range s.Operations {
		d, _, _ := db.Store.Get(op.Key)
		sim, err := op.Exec(d)
		if err != nil {
			db.Store.Unlock()
			return err
		}

		err = db.Check(s.Policy, op, sim)
		if err != nil {
			db.Store.Unlock()
			return err
		}
	}
	db.Store.Unlock()

	// Promise: Check for conflicts with staging
	db.stagingMutex.RLock()
	defer db.stagingMutex.RUnlock()
	for _, s2 := range db.staging {
		if s.CheckConflict(s2.spore) != nil {
			return ErrConflictingWithStaging
		}
	}

	return nil
}

// Submit broadcasts the Spore to the Mycelium, then tries to endorse it with current state.
func (db *DB) Submit(s *Spore) error {
	db.Messages <- s
	return db.Endorse(s)
}

// Endorse tries to endorse a Spore, calling CanEndorse before any operation.
// It either adds the Spore to the staging list, pending list or discards it.
func (db *DB) Endorse(s *Spore) error {
	err := db.CanEndorse(s)
	if err == ErrConflictingWithStaging {
		db.waitingMutex.Lock()
		db.waiting[s.Uuid] = &dbTrigger{
			spore: s,
		}
		db.waitingMutex.Unlock()
		return nil
	} else if err == nil {
		db.executeEndorsement(s)
		return nil
	}
	return err
}

func (db *DB) executeEndorsement(s *Spore) {
	signature, err := db.KeyRing.Sign(db.HashSpore(s))
	if err != nil {
		return // TODO log signature error
	}

	e := &Endorsement{
		Uuid:      s.Uuid,
		Emitter:   db.Identity,
		Signature: signature,
	}

	db.Messages <- e // Broadcast our endorsement for this spore

	// Is the policy only requires one endorsement, bypass staging list
	if db.policies[s.Policy].Quorum <= 1 {
		_ = db.Apply(s)
		return
	}

	db.stagingMutex.Lock()
	defer db.stagingMutex.Unlock()

	timer := time.AfterFunc(
		deadlineToDuration(s.Deadline),
		func() { db.gc <- s },
	)

	db.staging[s.Uuid] = &dbTrigger{
		timer:        timer,
		spore:        s,
		endorsements: []*Endorsement{e},
	}
}

// AddEndorsement registers the incoming endorsement.
func (db *DB) AddEndorsement(e *Endorsement) {
	db.addEndorsementMap(e, db.waiting, &db.waitingMutex)
	trigger := db.addEndorsementMap(e, db.staging, &db.stagingMutex)

	if trigger == nil {
		return
	}

	// Should we execute the spore?
	db.stagingMutex.Lock()
	defer db.stagingMutex.Unlock()

	policy := db.policies[trigger.spore.Policy]
	if policy.Quorum <= uint64(len(trigger.endorsements)) {
		trigger.timer.Stop()
		delete(db.staging, trigger.spore.Uuid)
		go func() { _ = db.Apply(trigger.spore) }()
	}
}

func (db *DB) addEndorsementMap(e *Endorsement, ma map[string]*dbTrigger, mu sync.Locker) *dbTrigger {
	mu.Lock()
	defer mu.Unlock()

	// Spore being processed?
	trigger, ok := ma[e.Uuid]
	if !ok {
		return nil // TODO retry till timeout or spore reception
	}

	// Already registered endorsement?
	for _, e2 := range trigger.endorsements {
		if e.Emitter == e2.Emitter {
			return nil
		}
	}

	hash := db.HashSpore(trigger.spore)
	err := db.KeyRing.Verify(e.Emitter, hash, e.Signature)
	if err != nil {
		return nil // TODO log verification error
	}

	trigger.endorsements = append(trigger.endorsements, e)
	return trigger
}

func deadlineToDuration(t *timestamp.Timestamp) time.Duration {
	if t == nil {
		return time.Hour
	}

	deadline := time.Unix(t.Seconds, int64(t.Nanos))
	return deadline.Sub(time.Now())
}
