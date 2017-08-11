package myc

import (
	"io"

	"gitlab.com/SporeDB/sporedb/myc/protocol"
)

type transport interface {
	io.Closer

	Bind(address string) (conn, error)
	Listen(address string, hook hookFn) error
}

type conn interface {
	protocol.Transport

	SetHandshake(func() error)
	Raw() protocol.Transport
}

type hookFn func(n *Peer, c conn)
