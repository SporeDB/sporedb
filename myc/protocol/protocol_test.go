package protocol

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Call_Pack(t *testing.T) {
	c := &Call{
		F: FnHELLO,
		M: &Hello{Version: 1},
	}

	data, err := c.Pack()
	require.Nil(t, err)
	require.Exactly(t, byte(0x01), data[0])

	l, n := binary.Uvarint(data[1:])
	require.Exactly(t, l, uint64(len(data[1+n:])))
}

func Test_Call_Unpack(t *testing.T) {
	c := &Call{
		F: FnHELLO,
		M: &Hello{Version: 5},
	}

	data, _ := c.Pack()
	buffer := bytes.NewBuffer(data)

	c2 := &Call{}
	err := c2.Unpack(buffer)
	require.Nil(t, err)
	require.IsType(t, c.M, c2.M, "must retrieve the same type")
	require.Exactly(t, c.M.(*Hello).Version, c2.M.(*Hello).Version)
}

func Test_Call_Unpack_Invalid(t *testing.T) {
	c := &Call{}
	require.NotNil(t, c.Unpack(bytes.NewBuffer([]byte{})), "must handle empty data")
	require.NotNil(t, c.Unpack(bytes.NewBuffer([]byte{0xf2})), "must handle invalid function")
	require.NotNil(t, c.Unpack(bytes.NewBuffer([]byte{0x01, 0xff})), "must handle invalid uvarint")
	require.NotNil(t, c.Unpack(bytes.NewBuffer([]byte{0x01, 0xff})), "must handle invalid uvarint")
	require.NotNil(t, c.Unpack(bytes.NewBuffer([]byte{0x01, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01})), "must handle too large uvarint")
	require.NotNil(t, c.Unpack(bytes.NewBuffer([]byte{0x01, 0x02, 0xff})), "must handle too small raw protobuf")
}
