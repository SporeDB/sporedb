package db

import (
	"fmt"
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

	type checkDoabilityCase struct {
		op          *Operation
		data        []byte
		errExpected bool
	}
	testCases := []checkDoabilityCase{
		{opSet, []byte("world"), false},
		{opSet, nil, false},
		{opAdd, []byte("2.5"), false},
		{opMul, []byte("2.5"), false},
		{opAdd, []byte{}, false},
		{opMul, []byte{}, false},
		{opAdd, nil, false},
		{opMul, nil, false},
		{opAdd, []byte("2.x"), true},
		{opMul, []byte("2.x"), true},
		{opBad, []byte("2.5"), true},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s %s", tc.op.Op.String(), tc.data), func(t *testing.T) {
			_, err := tc.op.CheckDoability(tc.data)
			if !tc.errExpected {
				require.Nil(t, err)
			} else {
				require.NotNil(t, err)
			}
		})
	}
}
