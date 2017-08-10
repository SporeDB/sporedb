package db

import (
	"io"
	"sync"

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
	// List returns the map of keys with their values.
	List() (map[string]*version.V, error)
}
