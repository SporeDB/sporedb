package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOperation_CheckConflict(t *testing.T) {
	ok := func(t *testing.T, a, b *Operation) {
		require.Nil(t, a.CheckConflict(b))
		require.Nil(t, b.CheckConflict(a))
	}
	ko := func(t *testing.T, a, b *Operation) {
		require.NotNil(t, a.CheckConflict(b))
		require.NotNil(t, b.CheckConflict(a))
	}

	t.Run("SET SET different key", func(t *testing.T) {
		op1 := &Operation{Key: "a", Op: Operation_SET, Data: []byte("hello")}
		op2 := &Operation{Key: "b", Op: Operation_SET, Data: []byte("world")}
		ok(t, op1, op2)
	})
	t.Run("SET SET same data", func(t *testing.T) {
		op1 := &Operation{Key: "c", Op: Operation_SET, Data: []byte("hello")}
		op2 := &Operation{Key: "c", Op: Operation_SET, Data: []byte("hello")}
		ok(t, op1, op2)
	})
	t.Run("SET SET", func(t *testing.T) {
		op1 := &Operation{Key: "d", Op: Operation_SET, Data: []byte("hello")}
		op2 := &Operation{Key: "d", Op: Operation_SET, Data: []byte("world")}
		ko(t, op1, op2)
	})
	t.Run("SET ADD", func(t *testing.T) {
		op1 := &Operation{Key: "e", Op: Operation_SET, Data: []byte{0x01}}
		op2 := &Operation{Key: "e", Op: Operation_ADD, Data: []byte{0x02}}
		ko(t, op1, op2)
	})
	t.Run("ADD ADD", func(t *testing.T) {
		op1 := &Operation{Key: "f", Op: Operation_ADD, Data: []byte{0x01}}
		op2 := &Operation{Key: "f", Op: Operation_ADD, Data: []byte{0x02}}
		ok(t, op1, op2)
	})
}

func TestOperation_CheckDoability(t *testing.T) {
	opSet := &Operation{Op: Operation_SET, Data: []byte("hello")}
	opAdd := &Operation{Op: Operation_ADD, Data: []byte("1.5")}
	opMul := &Operation{Op: Operation_MUL, Data: []byte("1.000000001e-22")}
	opBad := &Operation{Op: Operation_MUL, Data: []byte("bad")}

	t.Run("RAW", func(t *testing.T) {
		require.Nil(t, opSet.CheckDoability([]byte("world")))
		require.Nil(t, opSet.CheckDoability(nil))
	})

	t.Run("NUMERIC CHECK TYPE", func(t *testing.T) {
		require.Nil(t, opAdd.CheckDoability([]byte("2.5")))
		require.Nil(t, opMul.CheckDoability([]byte("2.5")))
		require.Nil(t, opAdd.CheckDoability([]byte{}))
		require.Nil(t, opMul.CheckDoability([]byte{}))
		require.Nil(t, opAdd.CheckDoability(nil))
		require.Nil(t, opMul.CheckDoability(nil))
		require.NotNil(t, opAdd.CheckDoability([]byte("2.x")))
		require.NotNil(t, opMul.CheckDoability([]byte("2.x")))
		require.NotNil(t, opBad.CheckDoability([]byte("2.5")))
	})
}
