package db

import (
	"errors"
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
		err := op.CheckDoability(d)
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

// Endorse tries to endorse a Spore, calling CanEndorse before any operation.
// It either adds the Spore to the staging list, pending list or discards it.
func (db *DB) Endorse(s *Spore) *Endorsement {
	err := db.CanEndorse(s)
	if err == ErrConflictingWithStaging {
		db.waitingMutex.Lock()
		c := make(chan *Endorsement, 1)
		db.waiting[s.Uuid] = &dbTrigger{
			channel: c,
			spore:   s,
		}
		db.waitingMutex.Unlock()
		return <-c
	} else if err == nil {
		return db.executeEndorsement(s)
	}
	return nil
}

func (db *DB) executeEndorsement(s *Spore) *Endorsement {
	db.stagingMutex.Lock()
	defer db.stagingMutex.Unlock()

	timer := time.AfterFunc(
		deadlineToDuration(s.Deadline),
		func() { db.gc <- s },
	)

	db.staging[s.Uuid] = &dbTrigger{
		timer: timer,
		spore: s,
	}

	return &Endorsement{}
}

func deadlineToDuration(t *timestamp.Timestamp) time.Duration {
	deadline := time.Unix(t.Seconds, int64(t.Nanos))
	return deadline.Sub(time.Now())
}
