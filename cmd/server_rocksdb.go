// +build rocksdb

package cmd

import (
	"gitlab.com/SporeDB/sporedb/db"
	"gitlab.com/SporeDB/sporedb/db/drivers/rocksdb"
)

func init() {
	addDriver("rocksdb", func(p string) (db.Store, error) {
		return rocksdb.New(p)
	})
}
