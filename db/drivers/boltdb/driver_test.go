package boltdb

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/SporeDB/sporedb/db/version"
)

var ts *S

func TestMain(m *testing.M) {
	path, err := ioutil.TempDir("", "sporedb_boltdb_")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ts, err = New(filepath.Join(path, "db"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	res := m.Run()

	_ = ts.Close()
	_ = os.RemoveAll(path)
	os.Exit(res)
}

func TestS_PutGet(t *testing.T) {
	k := "testSet"
	cases := [][]byte{
		[]byte("Hello world!"),
		[]byte{},
		make([]byte, 4*1024*1024),
	}

	for _, d := range cases {
		v := version.New(d)
		err := ts.Set(k, d, v)
		require.Nil(t, err)

		d2, v2, err := ts.Get(k)
		require.Nil(t, err)
		require.Exactly(t, d, d2)
		require.Exactly(t, v, v2)
	}
}

func TestS_PutGetBatch(t *testing.T) {
	keys := []string{
		"testBatch_a",
		"testBatch_b",
		"testBatch_c",
	}
	values := [][]byte{
		[]byte("Hello"),
		[]byte("World!"),
		[]byte{},
	}

	versions := make([]*version.V, len(keys))
	for i, v := range values {
		versions[i] = version.New(v)
	}

	require.Nil(t, ts.SetBatch(keys, values, versions))
	for i, k := range keys {
		value, v, err := ts.Get(k)
		require.Nil(t, err)
		require.Nil(t, v.Matches(versions[i]))
		require.Exactly(t, values[i], value)
	}
}

func TestS_Get_Unknown(t *testing.T) {
	_, v, err := ts.Get("testUnknown")
	require.NotNil(t, err)
	require.Exactly(t, v, version.NoVersion)
}
