// Package drivers holds required constructor for database drivers.
package drivers

import "gitlab.com/SporeDB/sporedb/db"

// Constructor is the mandatory prototype of each driver's constructor.
type Constructor func(path string) (store db.Store, err error)
