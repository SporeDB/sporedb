package operations

func setGeneric(input []byte, current *Value, add bool) error {
	s, err := current.Set()
	if err != nil {
		return ErrNotValidSet
	}

	if add {
		_, err = s.Add(input)
	} else {
		_, err = s.Remove(input)
	}

	if err != nil {
		return err
	}
	current.Raw, err = s.MarshalBinary()
	return err
}

// Sadd adds the input to the current set.
func Sadd(input []byte, current *Value) error {
	return setGeneric(input, current, true)
}

// Srem removes the input form the current set.
func Srem(input []byte, current *Value) error {
	return setGeneric(input, current, false)
}
