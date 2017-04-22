package db

import (
	"bytes"
	"errors"
	"regexp"

	"gitlab.com/SporeDB/sporedb/db/operations"
)

// Error messages for policy.
var (
	ErrUnknownPolicy = errors.New("the requested policy is unknown")
	ErrOpTooLarge    = errors.New("the requested operation is too large for the policy")
	ErrOpNotAllowed  = errors.New("the requested operation is not allowed by the policy")
	ErrOpDisabledKey = errors.New("the requested key is not modifiable according to the policy")
)

// NonePolicy is a basic policy used for testing and development.
var NonePolicy = &Policy{
	Uuid:    "none",
	Comment: "Allows everything on every key. Should only used for testing purposes.",
	Specs: []*OSpec{{
		Key: &OSpec_Regex{".*"},
	}},
}

// Check checks that a given operation is valid given its simulation and its database policy.
func (db *DB) Check(policy string, o *Operation, value *operations.Value) error {
	p := db.policies[policy]
	if p == nil {
		return ErrUnknownPolicy
	}

	// TODO check global size

	// Check simulation size
	l := uint64(len(value.Raw))
	if p.MaxOpSize > 0 && l > p.MaxOpSize {
		return ErrOpTooLarge
	}

	var valid bool
	for i, s := range p.Specs {
		if db.policiesReg[policy][i].MatchString(o.Key) {
			if s.MaxSize > 0 && l > s.MaxSize {
				return ErrOpTooLarge
			}
			if err := s.checkOp(o); err != nil {
				return err
			}
			valid = true
		}
	}

	if !valid {
		return ErrOpDisabledKey
	}
	return nil
}

func (p *Policy) compileRegexes() (r []*regexp.Regexp, err error) {
	for _, s := range p.Specs {
		if n := s.GetName(); n != "" {
			r = append(r, regexp.MustCompile(regexp.QuoteMeta(n)))
		} else if rs := s.GetRegex(); rs != "" {
			var reg *regexp.Regexp
			reg, err = regexp.Compile(rs)
			if err != nil {
				return
			}
			r = append(r, reg)
		} else {
			err = errors.New("policy contains invalid key specification")
			return
		}
	}
	return
}

func (p *Policy) pubToEndorser(pub []byte) *Endorser {
	if p == nil {
		return nil
	}

	for _, e := range p.Endorsers {
		if bytes.Equal(e.Public, pub) {
			return e
		}
	}

	return nil
}

func (s *OSpec) checkOp(o *Operation) error {
	if len(s.AllowedOperations) == 0 {
		return nil // everything is allowed
	}

	for _, ao := range s.AllowedOperations {
		if o.Op == ao {
			return nil // found operation in allowed operations
		}
	}

	return ErrOpNotAllowed
}
