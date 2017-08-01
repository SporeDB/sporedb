package myc

import (
	"errors"
	"io"
	"net"
	"time"

	"gitlab.com/SporeDB/sporedb/myc/protocol"
)

type transportTCP struct {
	listener *net.TCPListener
}

func (t *transportTCP) Bind(address string) (conn, error) {
	a, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return nil, err
	}

	c := &connTCP{
		connChan: make(chan error),
		errChan:  make(chan error),
	}

	go c.Connect(a)
	return c, <-c.connChan
}

func (t *transportTCP) Listen(address string, hook hookFn) error {
	a, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return err
	}

	t.listener, err = net.ListenTCP("tcp", a)
	if err != nil {
		return err
	}

	for {
		conn, err := t.listener.AcceptTCP()
		if err != nil {
			return err
		}

		c := &connTCP{
			TCPConn:  conn,
			connChan: make(chan error),
			errChan:  make(chan error),
		}

		// Start watch routine
		go func(c *connTCP) {
			err, ok := <-c.errChan
			if ok {
				c.connChan <- err
			}
		}(c)

		// Start hook routine
		go hook(&Peer{
			Node: protocol.Node{Address: c.RemoteAddr().String()},
			conn: c,
		})
	}
}

func (t *transportTCP) Close() error {
	return t.listener.Close()
}

type connTCP struct {
	*net.TCPConn

	handshake         func() error
	connChan, errChan chan error
}

func (c *connTCP) SetHandshake(f func() error) {
	c.handshake = f
}

func (c *connTCP) Close() error {
	close(c.errChan)
	close(c.connChan)
	if c.TCPConn != nil {
		return c.TCPConn.Close()
	}
	return nil
}

func (c *connTCP) Connect(address *net.TCPAddr) {
	var err error
	for open := true; open; _, open = <-c.errChan {
		c.TCPConn, err = net.DialTCP("tcp", nil, address)
		c.connChan <- nil
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		if c.handshake != nil {
			_ = c.handshake()
		}
	}
}

func (c *connTCP) ReadByte() (byte, error) {
	d := make([]byte, 1)
	_, err := io.ReadFull(c, d)
	return d[0], err
}

func (c *connTCP) Read(b []byte) (int, error) {
	var n int
	var err error
	if c.TCPConn == nil {
		n, err = 0, errors.New("no connection established")
	} else {
		n, err = c.TCPConn.Read(b)
	}
	if err != nil {
		c.errChan <- err
		err = <-c.connChan
	}

	return n, err
}

func (c *connTCP) Write(b []byte) (int, error) {
	var n int
	var err error
	if c.TCPConn == nil {
		n, err = 0, errors.New("no connection established")
	} else {
		n, err = c.TCPConn.Write(b)
	}
	if err != nil {
		c.errChan <- err
		err = <-c.connChan
	}

	return n, err
}
