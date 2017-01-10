package db

import (
	"bytes"
	"errors"
)

// ParallelMatrix is used to know which operation can be run in parallel on a specific object.
var ParallelMatrix = map[Operation_Op]bool{
	Operation_ADD: true,
	Operation_MUL: true,
}

// CheckConflict returns an error if two operations cannot be executed in parallel.
func (o *Operation) CheckConflict(o2 *Operation) error {
	if o.Key != o2.Key {
		return nil
	}

	if o.Op != o2.Op {
		return errors.New("operation mismatch")
	}

	// Special cases
	if o.Op == Operation_SET && bytes.Equal(o.Data, o2.Data) {
		return nil
	}

	// General case
	if !ParallelMatrix[o.Op] {
		return errors.New("non-parallel operation " + o.Op.String())
	}

	return nil
}

// Exec returns the result of the given operation against stored data.
func (o *Operation) Exec(data []byte) (result []byte, err error) {
	r, implemented := runners[o.Op]
	if !implemented {
		err = errors.New("operation not yet implemented")
		return
	}

	result, err = r(o.Data, data)
	return
}
