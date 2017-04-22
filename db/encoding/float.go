package encoding

import "math/big"

// Float holds a arbitrary-precision float, interally backed by Go's big.float.
type Float struct {
	*big.Float
}

// NewFloat returns a new float with 0 value.
func NewFloat() *Float {
	return &Float{Float: big.NewFloat(0)}
}

// MarshalBinary returns a human-readable representation of a float.
func (f *Float) MarshalBinary() (data []byte, err error) {
	return f.MarshalText()
}

// UnmarshalBinary parses a human-readable representation of a float.
func (f *Float) UnmarshalBinary(data []byte) error {
	if len(data) == 0 {
		f.Float = big.NewFloat(0)
		return nil
	}

	return f.UnmarshalText(data)
}

// Add returns a new Float from the addition of f and g.
func (f *Float) Add(g *Float) *Float {
	bf := new(big.Float).Add(f.Float, g.Float)
	return &Float{Float: bf}
}

// Mul returns a new Float from the multiplication of f and g.
func (f *Float) Mul(g *Float) *Float {
	bf := new(big.Float).Mul(f.Float, g.Float)
	return &Float{Float: bf}
}
