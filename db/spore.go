package db

import (
	"errors"
	"time"
)

// ErrDeadlineExpired is returned when a spore cannot be endorsed due to an expired deadline according to local clock.
var ErrDeadlineExpired = errors.New("unable to endorse a spore with expired deadline")

// ErrConflictingWithStaging is returned when a node cannot endorse a spore because it already has endorsed a conflicting spore.
var ErrConflictingWithStaging = errors.New("unable to endorse a spore due to conflicting promise")

// CanEndorse TODO
func CanEndorse(s *Spore, store Store, staging []*Spore) error {
	// Timeout: Check deadline
	if !s.checkDeadline() {
		return ErrDeadlineExpired
	}

	// Consistency: Check that the operations are no behind the state and fulfill the types
	store.Lock()

	for k, v := range s.Requirements {
		_, v2, err := store.Get(k)
		if err != nil {
			return err
		}

		if v2.Matches(v) != nil {
			return err
		}
	}

	for _, op := range s.Operations {
		d, _, _ := store.Get(op.Key)
		err := op.CheckDoability(d)
		if err != nil {
			return err
		}
	}

	store.Unlock()

	// Promise: Check for conflicts with staging
	for _, s2 := range staging {
		if s.CheckConflict(s2) != nil {
			return ErrConflictingWithStaging
		}
	}

	return nil
}

func (s *Spore) checkDeadline() bool {
	return s.Deadline.Seconds >= time.Now().Unix()
}

// CheckConflict returns an error if two spores are conflicting.
func (s *Spore) CheckConflict(s2 *Spore) error {
	if s.Policy != s2.Policy {
		return nil
	}

	for _, op := range s.Operations {
		for _, op2 := range s2.Operations {
			if err := op.CheckConflict(op2); err != nil {
				return err
			}
		}
	}

	return nil
}
