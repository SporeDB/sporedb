package operations

// Set sets the output value to the input value raw data.
func Set(input []byte, current *Value) error {
	current.reset()
	current.Raw = input
	return nil
}

// Append appends the raw input to the current value.
func Append(input []byte, current *Value) error {
	current.reset()
	current.Raw = append(current.Raw, input...)
	return nil
}
