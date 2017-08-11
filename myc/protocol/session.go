package protocol

import (
	"bytes"
	"errors"
	"io"
)

// Errors for session management.
var (
	ErrOldTimestamp     = errors.New("session timestamp too old")
	ErrInvalidPublicKey = errors.New("invalid public key")
)

// Session shall be used to establish a secure channel between
// two peers. It shall act as a proxy between the application
// and the underlying Transport.
type Session interface {
	// A Session can acts as a Transport itself, being transparent for the application.
	Transport

	// Hello builds a new Hello message (handshake).
	// It might be called several times, in case of connection reset.
	Hello() (*Hello, error)

	// Verify verifies Hello messages for conformity.
	// It might be called several times, in case of connection reset.
	Verify(*Hello) error

	// Open MUST be called after sending an Hello message and having received
	// a verified Hello message from the peer. It opens incoming and outgoing
	// encrypted channel.
	Open(Transport) error

	// IsTrusted returns weither the peer shall be trusted (is correctly authenticated).
	IsTrusted() bool
}

// Transport is a generic representation of a communication channel.
type Transport interface {
	io.ReadWriteCloser
	io.ByteReader
}

// NewLocalTransport returns two bounded Transport for use in tests.
func NewLocalTransport() (a, b Transport) {
	bufferA := &bytes.Buffer{}
	bufferB := &bytes.Buffer{}

	a = &testTransport{
		r: bufferA,
		w: bufferB,
	}

	b = &testTransport{
		r: bufferB,
		w: bufferA,
	}

	return
}

type testTransport struct {
	r *bytes.Buffer
	w *bytes.Buffer
}

func (t *testTransport) Read(p []byte) (n int, err error) {
	return t.r.Read(p)
}

func (t *testTransport) ReadByte() (byte, error) {
	return t.r.ReadByte()
}

func (t *testTransport) Write(p []byte) (n int, err error) {
	return t.w.Write(p)
}

func (t *testTransport) Close() error {
	return nil
}
