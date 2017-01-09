package db

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/SporeDB/sporedb/db/drivers/rocksdb"
)

func getTestingDB(t *testing.T) (db *DB, done func()) {
	path, err := ioutil.TempDir("", "sporedb_db_")
	require.Nil(t, err)

	store, err := rocksdb.New(path)
	require.Nil(t, err)

	db = NewDB(store)
	require.Nil(t, db.AddPolicy(NonePolicy))

	done = func() {
		_ = store.Close()
		_ = os.RemoveAll(path)
	}
	return
}

func TestDeadlineToDuration(t *testing.T) {
	s := NewSpore()
	s.SetTimeout(time.Second)
	require.InDelta(t, int64(time.Second), int64(deadlineToDuration(s.Deadline)), float64(time.Millisecond))
}

func TestDB_Single(t *testing.T) {
	db, done := getTestingDB(t)
	ch := make(chan bool)
	defer func() {
		<-ch // wait for goroutine termination
		done()
	}()

	db.Start(false)

	a := NewSpore()
	a.SetTimeout(200 * time.Millisecond)
	a.Operations = []*Operation{{
		Key:  "keyA",
		Op:   Operation_SET,
		Data: []byte("Hello"),
	}}

	b := NewSpore()
	b.SetTimeout(time.Second)
	b.Operations = []*Operation{{
		Key:  "keyB",
		Op:   Operation_SET,
		Data: []byte("Hello"),
	}}

	c := NewSpore()
	c.SetTimeout(time.Second)
	c.Operations = []*Operation{{
		Key:  "keyA",
		Op:   Operation_SET,
		Data: []byte("World"),
	}}

	go func() {
		db.Endorse(a)
		db.Endorse(c)
		ch <- true
	}()

	require.NotNil(t, db.Endorse(b))

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
