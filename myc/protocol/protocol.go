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
	"reflect"

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
	FnHELLO          Function = 0x01
	FnSPORE                   = 0x02
	FnENDORSE                 = 0x03
	FnRECOVERREQUEST          = 0x04
	FnRAW                     = 0x05
	FnGOSSIP                  = 0x06
	FnNODES                   = 0x07
)

var fnTypes = map[Function]reflect.Type{
	FnHELLO:          reflect.TypeOf(Hello{}),
	FnSPORE:          reflect.TypeOf(db.Spore{}),
	FnENDORSE:        reflect.TypeOf(db.Endorsement{}),
	FnRECOVERREQUEST: reflect.TypeOf(db.RecoverRequest{}),
	FnRAW:            reflect.TypeOf(Raw{}),
	FnGOSSIP:         reflect.TypeOf(Gossip{}),
	FnNODES:          reflect.TypeOf(Nodes{}),
}

var fnString = map[Function]string{
	FnSPORE:          "spore",
	FnENDORSE:        "endorse",
	FnRECOVERREQUEST: "recover",
	FnRAW:            "raw",
	FnGOSSIP:         "gossip",
	FnNODES:          "nodes",
}

func (f Function) String() string {
	return fnString[f]
}

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
	err = c.setUnpackRecipient()
	if err != nil {
		return err
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

func (c *Call) setUnpackRecipient() error {
	t, ok := fnTypes[c.F]
	if !ok {
		return errors.New("invalid function")
	}

	c.M = reflect.New(t).Interface().(proto.Message)
	return nil
}
