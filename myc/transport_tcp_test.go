package myc

import (
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_transportTCP(t *testing.T) {
	srv := &transportTCP{}
	cli := &transportTCP{}

	p := "localhost:4300"

	// Start echo server
	go func() {
		_ = srv.Listen(p, func(n *Node) {
			b := make([]byte, 64)
			_, _ = n.conn.Read(b)
			_, _ = n.conn.Write(b[:10])

			// Simulate a temporary crash
			_ = n.conn.Close()
			_ = srv.Close()

			time.Sleep(500 * time.Millisecond)
			_ = srv.Listen(p, func(nn *Node) {
				_, _ = nn.conn.Write(b[10:])
			})
		})
	}()

	c, err := cli.Bind(p)
	require.Nil(t, err)
	require.NotNil(t, c)

	b := []byte("Hello World via TCP Transport")
	b2 := make([]byte, len(b))

	n, err := c.Write(b)
	require.Nil(t, err)
	require.Exactly(t, len(b), n)

	n, err = io.ReadFull(c, b2)
	require.Nil(t, err)
	require.Exactly(t, len(b), n)
	require.Exactly(t, b, b2)
}
