package rocksdb

import (
	"errors"
	"sync"

	"gitlab.com/SporeDB/sporedb/db/version"

	"github.com/tecbot/gorocksdb"
)

var ro = gorocksdb.NewDefaultReadOptions()
var wo = gorocksdb.NewDefaultWriteOptions()

var errNotFound = errors.New("key corrupted or unknown")

// S is the driver for the RocksDB store engine.
type S struct {
	sync.Mutex

	db *gorocksdb.DB
}

// New generates a new RocksDB store from the storage path.
func New(path string) (s *S, err error) {
	s = &S{}

	opts := gorocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	s.db, err = gorocksdb.OpenDb(opts, path)
	return
}

// Get returns the value and the version stored currently for the specified key.
func (s *S) Get(key string) (value []byte, v *version.V, err error) {
	data, err := s.db.Get(ro, []byte(key))
	if err != nil {
		return
	}

	if data.Size() < version.VersionBytes {
		err = errNotFound
		v = version.NoVersion
		return
	}

	value = data.Data()[version.VersionBytes:]
	v = &version.V{}
	err = v.UnmarshalBinary(data.Data()[:version.VersionBytes])

	return
}

// Set sets the value and the version that must be stored for the specified key.
func (s *S) Set(key string, value []byte, v *version.V) error {
	rawVersion, err := v.MarshalBinary()
	if err != nil {
		return nil
	}

	return s.db.Put(wo, []byte(key), append(rawVersion[:version.VersionBytes], value...))
}

// SetBatch executes the given "Set" operations in a atomic way.
func (s *S) SetBatch(keys []string, values [][]byte, versions []*version.V) error {
	if len(keys) != len(values) || len(values) != len(versions) {
		return errors.New("invalid arguments")
	}

	batch := gorocksdb.NewWriteBatch()
	for i, k := range keys {
		rv, err := versions[i].MarshalBinary()
		if err != nil {
			return err
		}

		batch.Put([]byte(k), append(rv[:version.VersionBytes], values[i]...))
	}

	return s.db.Write(wo, batch)
}

// Close should be used after using the RocksDB store.
func (s *S) Close() error {
	s.db.Close()
	return nil
}
