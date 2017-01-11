package db

import (
	"time"

	"gitlab.com/SporeDB/sporedb/db/version"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/satori/go.uuid"
)

// NewSpore instanciates a new spore.
func NewSpore() *Spore {
	s := &Spore{}
	s.Uuid = uuid.NewV4().String()
	s.Policy = "none"
	s.Requirements = make(map[string]*version.V)
	return s
}

// SetTimeout updates the deadline of the spore according to current time.
func (s *Spore) SetTimeout(t time.Duration) {
	deadline := time.Now().Add(t)
	s.Deadline = &timestamp.Timestamp{
		Seconds: deadline.Unix(),
		Nanos:   int32(deadline.Nanosecond()),
	}
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

func (s *Spore) checkDeadline() bool {
	if s.Deadline == nil {
		return true
	}

	return s.Deadline.Seconds >= time.Now().Unix()
}
