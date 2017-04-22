package operations

import "gitlab.com/SporeDB/sporedb/db/encoding"

func floatGeneric(input []byte, current *Value, add bool) error {
	a := encoding.NewFloat()
	b, err := current.Float()
	err2 := a.UnmarshalBinary(input)
	if err != nil || err2 != nil {
		return ErrNotNumeric
	}

	if add {
		current.vfloat = a.Add(b)
	} else {
		current.vfloat = a.Mul(b)
	}

	current.Raw, err = current.vfloat.MarshalText()
	return err
}

// Add adds the input as float to the current value.
func Add(input []byte, current *Value) error {
	return floatGeneric(input, current, true)
}

// Mul multiplies the input as float to the current value.
func Mul(input []byte, current *Value) error {
	return floatGeneric(input, current, false)
}
