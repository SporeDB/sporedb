package protocol

import "github.com/golang/protobuf/proto"

// GetMessage returns the message used in the signature of a raw message.
func (r Raw) GetMessage() []byte {
	r.Signature = nil
	m, _ := proto.Marshal(&r)
	return m
}
