package db

import (
	"bytes"
	"errors"
	"math/big"
)

// ParallelMatrix is used to know which operation can be run in parallel on a specific object.
var ParallelMatrix = map[Operation_Op]bool{
	Operation_ADD: true,
	Operation_MUL: true,
}

// NumericOperations is used to know which operation can only be executed against numeric objects.
var NumericOperations = map[Operation_Op]bool{
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

// CheckDoability verifies that a given operation is valid against a specific object.
func (o *Operation) CheckDoability(data []byte) (simulation []byte, err error) {
	if NumericOperations[o.Op] {
		// Check data
		f := &big.Float{}
		err = f.UnmarshalText(o.Data)
		if err != nil {
			err = errors.New("non-numeric data value")
			return
		}

		// Check stored value
		if len(data) > 0 {
			f = &big.Float{}
			err = f.UnmarshalText(data)
			if err != nil {
				err = errors.New("non-numeric stored value")
				return
			}
		}
	}

	return
}
