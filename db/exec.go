package db

// TODO refactor for more elegant operation handling.
// Some ideas:
// - Use encoding.BinaryMarshaler / Unmarshaler for generic handling
// - Pass temporary values (like floats or Set) for faster processing
import (
	"errors"
	"math/big"

	"gitlab.com/SporeDB/sporedb/db/encoding"
)

// Errors returned when an operation does not match stored datatype.
var (
	ErrNotNumeric  = errors.New("non-numeric value")
	ErrNotValidSet = errors.New("non-valid set")
)

// TODO move to encoding package
func getFloat(data []byte) (*big.Float, error) {
	if len(data) == 0 {
		return big.NewFloat(0), nil
	}

	f := &big.Float{}
	err := f.UnmarshalText(data)
	if err != nil {
		err = ErrNotNumeric
	}
	return f, err
}

func getTwoFloats(a, b []byte) (x, y *big.Float, err error) {
	x, err = getFloat(a)
	if err != nil {
		return
	}

	y, err = getFloat(b)
	return
}

func getSet(data []byte) (*encoding.Set, error) {
	s := encoding.NewSet()

	err := s.UnmarshalBinary(data)
	if err != nil {
		return nil, ErrNotValidSet
	}

	return s, nil
}

type runner func(in, sto []byte) (out []byte, err error)

var runners = map[Operation_Op]runner{
	Operation_SET: func(in, sto []byte) (out []byte, err error) {
		return in, nil
	},
	Operation_CONCAT: func(in, sto []byte) (out []byte, err error) {
		return append(sto, in...), nil
	},
	Operation_ADD: func(in, sto []byte) (out []byte, err error) {
		x, y, err := getTwoFloats(in, sto)
		if err != nil {
			return
		}
		out, err = x.Add(x, y).MarshalText()
		return
	},
	Operation_MUL: func(in, sto []byte) (out []byte, err error) {
		x, y, err := getTwoFloats(in, sto)
		if err != nil {
			return
		}
		out, err = x.Mul(x, y).MarshalText()
		return
	},
	Operation_SADD: func(in, sto []byte) (out []byte, err error) {
		s, err := getSet(sto)
		if err != nil {
			return
		}

		_, err = s.Add(in)
		if err != nil {
			return
		}

		return s.MarshalBinary()
	},
	Operation_SREM: func(in, sto []byte) (out []byte, err error) {
		s, err := getSet(sto)
		if err != nil {
			return
		}

		_, err = s.Remove(in)
		if err != nil {
			return
		}

		return s.MarshalBinary()
	},
}
