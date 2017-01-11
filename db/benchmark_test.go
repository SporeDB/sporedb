package db

import (
	"testing"
	"time"
)

func genericDBBench(b *testing.B, op Operation_Op) {
	db, done := getTestingDB(&testing.T{})
	defer done()

	db.Start(false)

	b.ResetTimer()
	t := 100 * time.Millisecond

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			i++
			s := NewSpore()
			s.SetTimeout(t)
			s.Operations = []*Operation{{
				Key:  "key",
				Op:   op,
				Data: []byte("1"),
			}}

			_ = db.Endorse(s)
		}
	})
}

func BenchmarkDB_NoConflict(b *testing.B) {
	genericDBBench(b, Operation_ADD)
}

func BenchmarkDB_Conflicting(b *testing.B) {
	genericDBBench(b, Operation_CONCAT)
}
