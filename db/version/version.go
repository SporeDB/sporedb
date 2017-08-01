// Package version holds spores version management internal logic.
package version

import (
	"bytes"
	"crypto/sha512"
	"errors"
)

// ErrVersionMismatch is returned when two versions are not matching.
var ErrVersionMismatch = errors.New("the stored version does not match with required version")

// NoVersion is the default version that should be returned when no version is available in one store for a specific key.
var NoVersion = &V{}

// VersionBytes is the space used by the version when marshalled.
const VersionBytes = sha512.Size

// New returns a new version from some data.
func New(data []byte) *V {
	h := sha512.Sum512(data)
	return &V{
		Hash: h[:],
	}
}

// Matches returns an error is two versions are not matching.
func (v *V) Matches(v2 *V) error {
	if v2 == nil {
		return errors.New("only accepts non-nil version")
	}

	if !bytes.Equal(v.Hash, v2.Hash) {
		return ErrVersionMismatch
	}
	return nil
}

// MarshalBinary converts the version to a VersionBytes-sized bytes slice.
func (v *V) MarshalBinary() (data []byte, err error) {
	return v.Hash, nil
}

// UnmarshalBinary converts the input to a version.
func (v *V) UnmarshalBinary(data []byte) error {
	v.Hash = make([]byte, len(data))
	copy(v.Hash, data)
	return nil
}
