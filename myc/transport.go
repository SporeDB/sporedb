package myc

import "io"

type transport interface {
	io.Closer

	Bind(address string) (conn, error)
	Listen(address string, hook hookFn) error
}

type conn interface {
	io.ReadWriteCloser
	io.ByteReader

	SetHandshake(func() error)
}

type hookFn func(n *Peer)
