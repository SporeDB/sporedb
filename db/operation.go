package db

import (
	"bytes"
	"errors"
)

// ParallelType specifies the various options available when specifiying a parallelizable operation.
type ParallelType int16

// Definition for ParallelType.
// Each flag may be combined using bitwise operators.
const (
	ParallelTypeDEFAULT           ParallelType = 0x01
	ParallelTypeDISALLOWDIFFERENT              = 0x02
	ParallelTypeDISALLOWEQUAL                  = 0x04
)

// ParallelMatrix is used to know which operation can be run in parallel on a specific object.
var ParallelMatrix = map[Operation_Op]map[Operation_Op]ParallelType{
	Operation_SET: {Operation_SET: ParallelTypeDISALLOWDIFFERENT},
	Operation_ADD: {Operation_ADD: ParallelTypeDEFAULT},
	Operation_MUL: {Operation_MUL: ParallelTypeDEFAULT},
	Operation_SADD: {
		Operation_SADD: ParallelTypeDEFAULT,
		Operation_SREM: ParallelTypeDISALLOWEQUAL,
	},
	Operation_SREM: {
		Operation_SREM: ParallelTypeDEFAULT,
		Operation_SADD: ParallelTypeDISALLOWEQUAL,
	},
}

// CheckConflict returns an error if two operations cannot be executed in parallel.
func (o *Operation) CheckConflict(o2 *Operation) error {
	err := errors.New("non-parallel operations " + o.Op.String() + " / " + o2.Op.String())
	if o.Key != o2.Key {
		return nil
	}

	if ParallelMatrix[o.Op] == nil {
		return err
	}

	t := ParallelMatrix[o.Op][o2.Op]
	if t == 0 {
		return err
	}

	if ParallelTypeDEFAULT&t > 0 {
		return nil // bypass further checks
	}

	equal := bytes.Equal(o.Data, o2.Data)
	if equal && ParallelTypeDISALLOWEQUAL&t > 0 {
		return err
	}

	if !equal && ParallelTypeDISALLOWDIFFERENT&t > 0 {
		return err
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
