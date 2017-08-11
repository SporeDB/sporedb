package myc

import (
	"errors"
	"io"
	"net"
	"time"

	"go.uber.org/zap"

	"gitlab.com/SporeDB/sporedb/myc/protocol"
)

type transportTCP struct {
	listener *net.TCPListener
}

func (t *transportTCP) Bind(address string) (conn, error) {
	c := &connTCP{
		connChan: make(chan error),
		errChan:  make(chan error),
	}

	go c.connect(address)
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

		p := &Peer{
			Node: protocol.Node{Address: c.RemoteAddr().String()},
		}

		// Start watch routine
		go func(c *connTCP) {
			err, ok := <-c.errChan
			if ok {
				zap.L().Warn("Disconnected",
					zap.String("transport", "tcp"),
					zap.String("address", address),
					zap.Error(err),
				)
				c.connChan <- err
				hook(p, nil)
			}
		}(c)

		// Start hook routine
		go hook(p, c)
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

func (c *connTCP) connect(address string) {
	var err error

	dialer := net.Dialer{
		DualStack: true,
		KeepAlive: 10 * time.Second,
	}

	var conn net.Conn

	for open := true; open; _, open = <-c.errChan {
		for {
			conn, err = dialer.Dial("tcp", address)
			if err != nil {
				zap.L().Warn("Disconnected",
					zap.String("transport", "tcp"),
					zap.String("address", address),
					zap.Error(err),
				)
				time.Sleep(time.Second)
			} else {
				break
			}
		}

		c.TCPConn, _ = conn.(*net.TCPConn)
		zap.L().Info("Connected",
			zap.String("transport", "tcp"),
			zap.String("address", address),
		)

		if c.handshake != nil {
			c.connChan <- c.handshake()
		} else {
			c.connChan <- nil
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

func (c *connTCP) Raw() protocol.Transport {
	return &rawTCPTransport{c.TCPConn}
}

type rawTCPTransport struct {
	*net.TCPConn
}

func (r *rawTCPTransport) ReadByte() (byte, error) {
	d := make([]byte, 1)
	_, err := io.ReadFull(r, d)
	return d[0], err
}
