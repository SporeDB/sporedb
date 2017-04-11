package db

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/SporeDB/sporedb/db/drivers/rocksdb"
	"gitlab.com/SporeDB/sporedb/myc/sec"
)

func getTestingDB(t *testing.T) (db *DB, done func()) {
	path, err := ioutil.TempDir("", "sporedb_db_")
	require.Nil(t, err)

	store, err := rocksdb.New(path)
	require.Nil(t, err)

	keyRing := sec.NewKeyRingEd25519()
	_ = keyRing.CreatePrivate("password")

	db = NewDB(store, "test", keyRing)
	require.Nil(t, db.AddPolicy(NonePolicy))

	done = func() {
		_ = store.Close()
		_ = os.RemoveAll(path)
	}
	return
}

func TestHashSpore(t *testing.T) {
	db, done := getTestingDB(t)
	defer done()

	s := NewSpore()
	s.SetTimeout(time.Minute)
	require.Exactly(t, db.HashSpore(s), db.HashSpore(s))
	require.NotEqual(t, db.HashSpore(s), db.HashSpore(NewSpore()))
}

func TestDeadlineToDuration(t *testing.T) {
	s := NewSpore()
	s.SetTimeout(time.Second)
	require.InDelta(t, int64(time.Second), int64(deadlineToDuration(s.Deadline)), float64(50*time.Millisecond))
}

func TestDB_Single_Quorum1(t *testing.T) {
	db, done := getTestingDB(t)
	defer done()

	db.Start(false)
	s := NewSpore()
	s.SetTimeout(200 * time.Millisecond)
	s.Operations = []*Operation{{
		Key:  "keyA",
		Op:   Operation_SET,
		Data: []byte("Hello"),
	}, {
		Key:  "keyB",
		Op:   Operation_ADD,
		Data: []byte("5.42"),
	}}
	s.Emitter = db.Identity
	s.Signature, _ = db.KeyRing.Sign(db.HashSpore(s))

	_ = db.Endorse(s)

	// Give some time
	time.Sleep(10 * time.Millisecond)

	value, _, err := db.Get("keyA")
	require.Nil(t, err)
	require.Exactly(t, []byte("Hello"), value)

	value, _, err = db.Get("keyB")
	require.Nil(t, err)
	require.Exactly(t, []byte("5.42"), value)
}

func TestDB_Single_Quorum2(t *testing.T) {
	db, done := getTestingDB(t)
	ch := make(chan bool)
	defer func() {
		<-ch // wait for goroutine termination
		done()
	}()

	// Change policy to force Quorum
	db.policies["none"].Quorum = 2

	db.Start(false)

	a := NewSpore()
	a.SetTimeout(200 * time.Millisecond)
	a.Operations = []*Operation{{
		Key:  "keyA",
		Op:   Operation_SET,
		Data: []byte("Hello"),
	}}
	a.Emitter = db.Identity
	a.Signature, _ = db.KeyRing.Sign(db.HashSpore(a))

	b := NewSpore()
	b.SetTimeout(time.Second)
	b.Operations = []*Operation{{
		Key:  "keyB",
		Op:   Operation_SET,
		Data: []byte("Hello"),
	}}
	b.Emitter = db.Identity
	b.Signature, _ = db.KeyRing.Sign(db.HashSpore(b))

	c := NewSpore()
	c.SetTimeout(time.Second)
	c.Operations = []*Operation{{
		Key:  "keyA",
		Op:   Operation_SET,
		Data: []byte("World"),
	}}
	c.Emitter = db.Identity
	c.Signature, _ = db.KeyRing.Sign(db.HashSpore(c))

	go func() {
		_ = db.Endorse(a)
		_ = db.Endorse(c)
		ch <- true
	}()

	_ = db.Endorse(b)

	// Check buffers after a short delay
	time.Sleep(10 * time.Millisecond)

	db.waitingMutex.RLock()
	db.stagingMutex.RLock()

	require.NotNil(t, db.staging[a.Uuid])
	require.NotNil(t, db.staging[b.Uuid])
	require.Exactly(t, a, db.staging[a.Uuid].spore)
	require.Exactly(t, b, db.staging[b.Uuid].spore)
	require.Nil(t, db.staging[c.Uuid])

	require.NotNil(t, db.waiting[c.Uuid])
	require.Exactly(t, c, db.waiting[c.Uuid].spore)
	require.Nil(t, db.waiting[a.Uuid])
	require.Nil(t, db.waiting[b.Uuid])

	db.stagingMutex.RUnlock()
	db.waitingMutex.RUnlock()

	// Check buffers after a long delay: a should have timed out
	time.Sleep(200 * time.Millisecond)

	db.waitingMutex.RLock()
	db.stagingMutex.RLock()

	require.NotNil(t, db.staging[c.Uuid])
	require.Exactly(t, c, db.staging[c.Uuid].spore)
	require.Nil(t, db.staging[a.Uuid])

	require.Nil(t, db.waiting[c.Uuid])

	db.stagingMutex.RUnlock()
	db.waitingMutex.RUnlock()
}
