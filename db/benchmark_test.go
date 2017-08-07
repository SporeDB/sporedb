package db

import "testing"

func genericDBBench(b *testing.B, op Operation_Op) {
	db, done := getTestingDB(&testing.T{})
	defer done()

	db.Start(false)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			i++
			s, sign := getTestSpore(db)
			s.Operations = []*Operation{{
				Key:  "key",
				Op:   op,
				Data: []byte("1"),
			}}
			sign()

			err := db.Endorse(s)
			if err != nil {
				panic(err)
			}
		}
	})
}

func BenchmarkDB_NoConflict(b *testing.B) {
	genericDBBench(b, Operation_ADD)
}

func BenchmarkDB_Conflicting(b *testing.B) {
	genericDBBench(b, Operation_CONCAT)
}
