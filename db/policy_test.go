package db

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func getTestSpore(db *DB) (s *Spore, sign func()) {
	s = NewSpore()
	s.SetTimeout(100 * time.Millisecond)
	s.Emitter = db.Identity
	sign = func() {
		db.cache.Purge()
		s.Signature = nil
		s.Signature, _ = db.KeyRing.Sign(db.HashSpore(s))
	}
	return
}

func TestDB_PolicySize(t *testing.T) {
	db, done := getTestingDB(t)
	defer done()

	// Change policy to force maximum sizes
	db.policies["none"].MaxSize = 3
	db.policies["none"].MaxOpSize = 2
	defer func() {
		db.policies["none"].MaxSize = 0
		db.policies["none"].MaxOpSize = 0
	}()

	db.Start(false)

	s, sign := getTestSpore(db)
	s.Operations = []*Operation{{
		Key:  "a",
		Op:   Operation_SET,
		Data: []byte("ABC"),
	}}
	sign()
	require.Exactly(t, ErrOpTooLarge, db.Endorse(s), "too large operations must be blocked")

	s.Operations[0].Data = []byte("AB")
	sign()
	require.Nil(t, db.Endorse(s), "operations that comply to maximum size must be allowed")

	s, sign = getTestSpore(db)
	s.Operations = []*Operation{{
		Key:  "b",
		Op:   Operation_SET,
		Data: []byte("CD"),
	}}
	sign()
	require.Exactly(t, ErrPolicyQuotaExceeded, db.Endorse(s), "operations exceeding policy quota must be blocked")

	s.Operations = append(s.Operations, &Operation{
		Key:  "a",
		Op:   Operation_SET,
		Data: []byte("A"),
	})
	sign()
	require.Nil(t, db.Endorse(s), "operations that comply to policy quota must be allowed")
}
