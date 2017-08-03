package db

import (
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"

	"gitlab.com/SporeDB/sporedb/db/operations"

	"github.com/golang/protobuf/ptypes/timestamp"
)

// Error messages
var (
	ErrDeadlineExpired        = errors.New("unable to endorse a spore with expired deadline")
	ErrConflictingWithStaging = errors.New("unable to endorse a spore due to conflicting promise")
	ErrBehindRequirement      = errors.New("unable to endorse a spore due to unfulfillable requirement")

	ErrNoRelatedSpore        = errors.New("unable to find related spore")
	ErrDuplicatedEndorsement = errors.New("duplicated endorsement")
	ErrUnallowedEndorser     = errors.New("unallowed endorser")

	ErrGracePeriodExpired    = errors.New("unable to apply a spore with expired grace period")
	ErrDuplicatedApplication = errors.New("duplicated application")
)

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

	values := make(map[string]*operations.Value)

	for _, op := range s.Operations {
		v, ok := values[op.Key]
		if !ok {
			d, _, _ := db.Store.Get(op.Key)
			values[op.Key] = operations.NewValue(d)
			v = values[op.Key]
		}

		err := op.Exec(v)
		if err != nil {
			db.Store.Unlock()
			return err
		}

		err = db.Check(s.Policy, op, v)
		if err != nil {
			db.Store.Unlock()
			return err
		}
	}
	db.Store.Unlock()

	// Promise: Check for conflicts with staging
	return db.checkConflictWithStaging(s)
}

func (db *DB) checkConflictWithStaging(s *Spore) error {
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
func (db *DB) Submit(s *Spore) (err error) {
	// Sign the spore before submission
	s.Emitter, s.Signature = db.Identity, nil // ensure nil signature before hash
	s.Signature, err = db.KeyRing.Sign(hashMessage(s))
	if err != nil {
		zap.L().Error("Unable to sign the spore",
			zap.String("uuid", s.Uuid),
			zap.String("step", "submission"),
			zap.Error(err),
		)
		return err
	}

	db.Messages <- s
	return db.Endorse(s)
}

// VerifySporeSignature verifies emitter's signature of the given spore.
// It is passed by value because this function require's spore alteration.
func (db *DB) VerifySporeSignature(s Spore) error {
	signature := s.Signature
	s.Signature = nil
	hash := hashMessage(&s)

	if s.Emitter == db.Identity {
		s.Emitter = ""
	}

	return db.KeyRing.Verify(s.Emitter, hash, signature)
}

// Endorse tries to endorse a Spore, calling CanEndorse before any operation.
// It either adds the Spore to the staging list, pending list or discards it.
func (db *DB) Endorse(s *Spore) error {
	err := db.VerifySporeSignature(*s)
	if err != nil {
		return err
	}

	err = db.CanEndorse(s)
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
	policy := db.policies[s.Policy]
	if policy.Quorum == 0 {
		_ = db.Apply(s)
		return
	}

	pub, _, _ := db.KeyRing.GetPublic("")
	endorser := policy.pubToEndorser(pub)
	var endorsements []*Endorsement

	if endorser != nil {
		signature, err := db.KeyRing.Sign(db.HashSpore(s))
		if err != nil {
			zap.L().Error("Unable to sign the spore",
				zap.String("uuid", s.Uuid),
				zap.String("step", "endorsement"),
				zap.Error(err),
			)
			return
		}

		e := &Endorsement{
			Uuid:      s.Uuid,
			Emitter:   db.Identity,
			Signature: signature,
		}

		db.Messages <- e // Broadcast our endorsement for this spore

		// If the policy only requires one endorsement, bypass staging list
		if db.policies[s.Policy].Quorum == 1 {
			_ = db.Apply(s)
			return
		}

		endorsements = []*Endorsement{e}
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
		endorsements: endorsements,
	}
}

// AddEndorsement registers the incoming endorsement.
func (db *DB) AddEndorsement(e *Endorsement) error {
	trigger, err := db.addEndorsementMap(e, db.staging, &db.stagingMutex)

	if err != nil && err != ErrNoRelatedSpore {
		return err
	}

	if trigger == nil {
		_, werr := db.addEndorsementMap(e, db.waiting, &db.waitingMutex)
		return werr
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

	return nil
}

func (db *DB) addEndorsementMap(e *Endorsement, ma map[string]*dbTrigger, mu sync.Locker) (*dbTrigger, error) {
	mu.Lock()
	defer mu.Unlock()

	// Spore being processed?
	trigger, ok := ma[e.Uuid]
	if !ok {
		return nil, ErrNoRelatedSpore
	}

	// Already registered endorsement?
	for _, e2 := range trigger.endorsements {
		if e.Emitter == e2.Emitter {
			return nil, ErrDuplicatedEndorsement
		}
	}

	// Known endorser?
	emitter := e.Emitter
	if e.Emitter == db.Identity {
		emitter = "" // local endorsement case
	}

	pub, _, err := db.KeyRing.GetPublic(emitter)
	if err != nil {
		zap.L().Warn("Invalid endorsement",
			zap.String("uuid", e.Uuid),
			zap.String("endorser", e.Emitter),
			zap.String("step", "public"),
			zap.Error(err),
		)
		return nil, err
	}

	// Allowed endorser?
	if db.policies[trigger.spore.Policy].pubToEndorser(pub) == nil {
		zap.L().Warn("Invalid endorsement",
			zap.String("uuid", e.Uuid),
			zap.String("endorser", e.Emitter),
			zap.String("step", "policy"),
			zap.Error(err),
		)
		return nil, ErrUnallowedEndorser
	}

	// Well-formed signature?
	if err = db.KeyRing.Verify(emitter, db.HashSpore(trigger.spore), e.Signature); err != nil {
		zap.L().Warn("Invalid endorsement",
			zap.String("uuid", e.Uuid),
			zap.String("endorser", e.Emitter),
			zap.String("step", "crypto"),
			zap.Error(err),
		)
		return nil, err
	}

	trigger.endorsements = append(trigger.endorsements, e)
	return trigger, nil
}

func deadlineToDuration(t *timestamp.Timestamp) time.Duration {
	if t == nil {
		return time.Hour
	}

	deadline := time.Unix(t.Seconds, int64(t.Nanos))
	return time.Until(deadline)
}
