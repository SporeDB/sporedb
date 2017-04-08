package encoding

import (
	"errors"
	"io"
)

// Error constants for Sets
var (
	ErrEmptyElement = errors.New("invalid empty element")
)

// Set holds a standard hash set, internally backed by Go's maps.
// This data-structure is optimized for fast insertions and marshalling,
// at some cost in memory usage.
//
// It is absolutely NOT thread-safe.
type Set struct {
	// Elements may be directly accessed in READ-ONLY mode with the Element attribute.
	// The value is only used internally and has no signification for external packages.
	Elements map[string]int
	raw      []byte
}

// NewSet returns a new empty Set.
func NewSet() *Set {
	return &Set{
		Elements: make(map[string]int),
	}
}

// Contains return wether or not a particular element is part of a Set,
// with a O(1) complexity.
func (s *Set) Contains(element []byte) bool {
	if len(element) == 0 {
		return false
	}

	_, ok := s.Elements[string(element)]
	return ok
}

// Add adds one element to a set with a O(1) complexity.
func (s *Set) Add(element []byte) (inserted bool, err error) {
	if len(element) == 0 {
		err = ErrEmptyElement
		return
	}

	str := string(element)
	if _, ok := s.Elements[str]; ok {
		return
	}

	s.Elements[str] = len(s.raw)
	s.raw = append(s.raw, uint64ToBytes(uint64(len(element)))...)
	s.raw = append(s.raw, element...)
	inserted = true
	return
}

// Remove removes one element from a set with a O(n) complexity.
func (s *Set) Remove(element []byte) (removed bool, err error) {
	l := len(element)
	if l == 0 {
		err = ErrEmptyElement
		return
	}

	l += 8

	str := string(element)
	p, ok := s.Elements[str]
	if !ok {
		return
	}

	delete(s.Elements, str)
	s.raw = append(s.raw[:p], s.raw[l+p:]...)

	for e, n := range s.Elements {
		if n > p {
			s.Elements[e] = n - l
		}
	}

	removed = true
	return
}

// MarshalBinary returns a binary representation of this set with a O(1) complexity.
func (s *Set) MarshalBinary() (data []byte, err error) {
	if s.raw == nil {
		s.raw = []byte{}
	}

	return s.raw, nil
}

// UnmarshalBinary parses a binary representation of this set with a O(n) complexity.
// Invalid representations may return an io.ErrUnexpectedEOF error code.
func (s *Set) UnmarshalBinary(data []byte) error {
	s.Elements = make(map[string]int)
	s.raw = data

	l := len(data)

	for i := 0; i < l; {
		if i+8 > l {
			return io.ErrUnexpectedEOF
		}

		length := int(bytesToUint64(data[i : i+8]))

		if i+8+length > l {
			return io.ErrUnexpectedEOF
		}

		str := string(data[i+8 : i+8+length])
		s.Elements[str] = i

		i += 8 + length
	}

	return nil
}
