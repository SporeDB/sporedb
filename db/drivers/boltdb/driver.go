// Package boltdb provides the (default) BoldDB database driver.
package boltdb

import (
	"errors"
	"sync"

	"gitlab.com/SporeDB/sporedb/db/version"

	"github.com/boltdb/bolt"
)

var bucketName = []byte("sporedb")
var errNotFound = errors.New("key corrupted or unknown")

// S is the driver for the BoltDB store engine.
type S struct {
	sync.Mutex

	db *bolt.DB
}

// New generates a new BoltDB store from the storage path.
func New(path string) (s *S, err error) {
	s = &S{}
	s.db, err = bolt.Open(path, 0600, nil)
	if err != nil {
		return
	}

	err = s.db.Update(func(tx *bolt.Tx) error {
		_, e := tx.CreateBucketIfNotExists(bucketName)
		return e
	})

	if err != nil {
		_ = s.Close()
	}

	return
}

// Get returns the value and the version stored currently for the specified key.
func (s *S) Get(key string) (value []byte, v *version.V, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)

		data := b.Get([]byte(key))
		if len(data) < version.VersionBytes {
			v = version.NoVersion
			return errNotFound
		}

		value = data[version.VersionBytes:]
		v = &version.V{}
		return v.UnmarshalBinary(data[:version.VersionBytes])
	})

	return
}

// Set sets the value and the version that must be stored for the specified key.
func (s *S) Set(key string, value []byte, v *version.V) error {
	return s.SetBatch([]string{key}, [][]byte{value}, []*version.V{v})
}

// SetBatch executes the given "Set" operations in a atomic way.
func (s *S) SetBatch(keys []string, values [][]byte, versions []*version.V) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)

		for i, k := range keys {
			rv, err := versions[i].MarshalBinary()
			if err != nil {
				return err
			}

			err = b.Put([]byte(k), append(rv[:version.VersionBytes], values[i]...))
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// List returns the map of keys with their values.
func (s *S) List() (map[string]*version.V, error) {
	catalog := make(map[string]*version.V)
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		c := b.Cursor()

		for k, d := c.First(); k != nil; k, d = c.Next() {
			if len(d) >= version.VersionBytes {
				v := &version.V{}
				if v.UnmarshalBinary(d[:version.VersionBytes]) == nil {
					catalog[string(k)] = v
				}
			}
		}

		return nil
	})

	return catalog, err
}

// Close should be used after using the RocksDB store.
func (s *S) Close() error {
	return s.db.Close()
}
