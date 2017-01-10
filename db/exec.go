package db

import (
	"errors"
	"math/big"
)

// ErrNotNumeric is returned when a numeric value is expected and the provided one cannot be parsed.
var ErrNotNumeric = errors.New("non-numeric value")

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
}
