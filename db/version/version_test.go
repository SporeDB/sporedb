package version

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestV_Matches(t *testing.T) {
	a := New([]byte("hello"))
	b := New([]byte("hello"))
	c := New([]byte("world"))

	require.Nil(t, a.Matches(b))
	require.Nil(t, b.Matches(a))

	require.Exactly(t, ErrVersionMismatch, a.Matches(c))
	require.Exactly(t, ErrVersionMismatch, c.Matches(a))

	require.NotNil(t, a.Matches(nil))
}

func TestV_Marshal(t *testing.T) {
	a := New([]byte("hello"))
	d, err := a.MarshalBinary()
	require.Nil(t, err)

	b := &V{}
	err = b.UnmarshalBinary(d)
	require.Nil(t, err)
	require.Nil(t, a.Matches(b))
}
