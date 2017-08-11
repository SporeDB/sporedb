package myc

import (
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_transportTCP(t *testing.T) {
	srv := &transportTCP{}
	srv2 := &transportTCP{}
	cli := &transportTCP{}

	p := "localhost:4300"

	// Start echo server
	go func() {
		_ = srv.Listen(p, func(n *Peer, c conn) {
			b := make([]byte, 64)
			_, _ = c.Read(b)
			_, _ = c.Write(b[:10])

			// Simulate a temporary crash
			_ = srv.Close()
			_ = c.Close()

			time.Sleep(3000 * time.Millisecond)
			_ = srv2.Listen(p, func(nn *Peer, cc conn) {
				_, _ = cc.Write(b[10:])
				_ = srv2.Close()
			})
		})
	}()

	c, err := cli.Bind(p)
	require.Nil(t, err)
	require.NotNil(t, c)

	b := []byte("Hello World via TCP Transport")
	b2 := make([]byte, len(b))

	// Wait a bit for server startup
	var n int
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
		n, err = c.Write(b)

		if n > 0 {
			require.Nil(t, err)
			require.Exactly(t, len(b), n)
			break
		}
	}

	require.True(t, n > 0, "Should successfully write at least one byte")

	n, err = io.ReadFull(c, b2)
	require.Nil(t, err)
	require.Exactly(t, len(b), n)
	require.Exactly(t, b, b2)
}
