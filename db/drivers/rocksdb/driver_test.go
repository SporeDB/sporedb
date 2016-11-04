package rocksdb

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/SporeDB/sporedb/db/version"
)

var ts *S

func TestMain(m *testing.M) {
	path, err := ioutil.TempDir("", "sporedb_rocksdb_")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ts, err = New(path)
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

func TestS_Get_Unknown(t *testing.T) {
	_, v, err := ts.Get("testUnknown")
	require.NotNil(t, err)
	require.Exactly(t, v, version.NoVersion)
}

func TestS_Get_Corrupted(t *testing.T) {
	k := "testCorrupted"
	cases := [][]byte{
		[]byte{0x00, 0x01, 0x02},
		[]byte{},
		nil,
	}

	for i, d := range cases {
		err := ts.db.Put(wo, []byte(k), d)
		require.Nil(t, err)

		_, v, err := ts.Get(k)
		require.NotNil(t, err, fmt.Sprintf("case %d is not returning an error despite the corruption", i))
		require.Exactly(t, version.NoVersion, v, fmt.Sprintf("case %d is not returning NoVersion despite the corruption", i))
	}
}
