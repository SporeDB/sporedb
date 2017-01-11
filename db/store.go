package db

import (
	"io"
	"io/ioutil"
	"os"
	"sync"

	"gitlab.com/SporeDB/sporedb/db/drivers/rocksdb"
	"gitlab.com/SporeDB/sporedb/db/version"
)

// Store is the interface internal drivers must implement.
type Store interface {
	sync.Locker
	io.Closer
	// Get returns the value and the version stored currently for the specified key.
	Get(key string) (value []byte, version *version.V, err error)
	// Set sets the value and the version that must be stored for the specified key.
	Set(key string, value []byte, version *version.V) error
	// SetBatch executes the given "Set" operations in a atomic way.
	SetBatch(keys []string, values [][]byte, versions []*version.V) error
}

// GetTestStore returns a default store for testing purposes.
// The done return value must be called after running the tests to clean temporary files.
func GetTestStore() (s Store, done func(), err error) {
	path, err := ioutil.TempDir("", "rocksdb_")
	if err != nil {
		return
	}

	s, err = rocksdb.New(path)
	done = func() {
		if s != nil {
			_ = s.Close()
		}
		_ = os.RemoveAll(path)
	}

	return
}
