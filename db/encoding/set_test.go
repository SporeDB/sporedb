package encoding

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSet_Add(t *testing.T) {
	s := NewSet()
	e1 := []byte("alice")
	e2 := []byte{0x41, 0x00, 0x42}

	_, err := s.Add([]byte{})
	require.NotNil(t, err)

	_, err = s.Add(nil)
	require.NotNil(t, err)

	inserted, err := s.Add(e1)
	require.Nil(t, err)
	require.True(t, inserted)

	inserted, err = s.Add(e2)
	require.Nil(t, err)
	require.True(t, inserted)

	inserted, err = s.Add(e1)
	require.Nil(t, err)
	require.False(t, inserted, "should not insert already inserted data")

	require.Exactly(t, map[string]int{
		string(e1): 0,
		string(e2): 8 + 5,
	}, s.Elements)
	require.Exactly(t, []byte{0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 'a', 'l', 'i', 'c', 'e', 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x41, 0x00, 0x42}, s.raw)
}

// return a set with 3 elements:
// - 0x0a0b
// - 0x0c0d0e
// - 0x0f
func getTestSet() (*Set, [][]byte) {
	p := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	elements := [][]byte{
		{0x0a, 0x0b},
		{0x0c, 0x0d, 0x0e},
		{0x0f},
	}

	s := NewSet()
	for _, e := range elements {
		s.Elements[string(e)] = len(s.raw)
		s.raw = append(s.raw, byte(len(e)))
		s.raw = append(s.raw, p...)
		s.raw = append(s.raw, e...)
	}

	return s, elements
}

func TestSet_Remove(t *testing.T) {
	type removeCase struct {
		name             string
		data             []byte
		elementsExpected map[string]int
		rawExpected      []byte
		removedExpected  bool
		errExpected      bool
	}

	rs, e := getTestSet()
	str0, str1, str2 := string(e[0]), string(e[1]), string(e[2])
	raw0 := []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0a, 0x0b}
	raw1 := []byte{0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0c, 0x0d, 0x0e}
	raw2 := []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0f}

	testCases := []removeCase{
		{"remove unknown", []byte("unknown"), rs.Elements, rs.raw, false, false},
		{"remove empty", []byte{}, nil, nil, false, true},
		{"remove nil", nil, nil, nil, false, true},
		{"remove last", e[2], map[string]int{str0: 0, str1: len(raw0)}, append(raw0, raw1...), true, false},
		{"remove middle", e[1], map[string]int{str0: 0, str2: len(raw0)}, append(raw0, raw2...), true, false},
		{"remove first", e[0], map[string]int{str1: 0, str2: len(raw1)}, append(raw1, raw2...), true, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := getTestSet()
			removed, err := s.Remove(tc.data)
			require.Exactly(t, tc.removedExpected, removed, "invalid removed value")

			if tc.errExpected {
				require.NotNil(t, err, "expected error")
				return
			}
			require.Nil(t, err, "unexpected error")
			require.Exactly(t, tc.rawExpected, s.raw, "wrong raw value")
			require.Exactly(t, tc.elementsExpected, s.Elements, "wrong elements value")
		})
	}
}

func TestSet_Marshalling(t *testing.T) {
	s, _ := getTestSet()
	data, err := s.MarshalBinary()
	require.Nil(t, err)
	require.Exactly(t, s.raw, data, "should be equal to the raw data")

	s2 := NewSet()
	err = s2.UnmarshalBinary(data)
	require.Nil(t, err)
	require.Exactly(t, s.raw, s2.raw)
	require.Exactly(t, s.Elements, s2.Elements)

	snil := NewSet()
	err = snil.UnmarshalBinary(nil)
	require.Nil(t, err)
}

func TestSet_Contains(t *testing.T) {
	s, e := getTestSet()
	for _, ee := range e {
		require.True(t, s.Contains(ee))
	}
	require.False(t, s.Contains([]byte("invalid")))
	require.False(t, s.Contains(nil))
}
