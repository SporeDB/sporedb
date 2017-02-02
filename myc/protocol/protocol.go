// Package protocol holds the SporeDB mycelium protocol.
//
// Paquet format:
// - 1 byte for function selection
// - n bytes for data length specification (uvarint)
// - remaining bytes containing data
package protocol

import (
	"encoding/binary"
	"errors"
	"io"

	"gitlab.com/SporeDB/sporedb/db"

	"github.com/golang/protobuf/proto"
)

// Version is the current version of the protocol.
// Two different versions are not supposed to be able to communicate.
const Version uint64 = 1

// Function represents the content of a package.
type Function byte

// Function values.
const (
	FnHELLO   Function = 0x01
	FnSPORE            = 0x02
	FnENDORSE          = 0x03
)

// Call represents a package that can be sent across the mycelium network.
type Call struct {
	F Function
	M proto.Message
}

// Pack generates a ready-to-send package for the Call.
func (c *Call) Pack() (data []byte, err error) {
	// Generate protobuf wire data
	raw, err := proto.Marshal(c.M)
	if err != nil {
		return
	}

	// Make a arbitrary size data buffer
	data = make([]byte, 1+binary.MaxVarintLen64)
	data[0] = byte(c.F)
	n := binary.PutUvarint(data[1:], uint64(len(raw)))

	// Add raw data
	data = append(data[:n+1], raw...)
	return
}

// InputStream represents a reader that can also be read byte by byte.
type InputStream interface {
	io.Reader
	io.ByteReader
}

// Unpack retrieves one Call from raw stream.
func (c *Call) Unpack(in InputStream) error {
	// Read function
	b, err := in.ReadByte()
	if err != nil {
		return err
	}

	c.F = Function(b)

	switch c.F {
	case FnHELLO:
		c.M = &Hello{}
	case FnSPORE:
		c.M = &db.Spore{}
	case FnENDORSE:
		c.M = &db.Endorsement{}
	default:
		return errors.New("invalid function")
	}

	// Read length
	l, err := binary.ReadUvarint(in)
	if err != nil {
		return err
	}

	// Unmarshal data
	i := int(l)
	if i < 0 {
		return errors.New("invalid length")
	}

	buf := make([]byte, i)
	_, err = io.ReadFull(in, buf)
	if err != nil {
		return err
	}

	return proto.Unmarshal(buf, c.M)
}