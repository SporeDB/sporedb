// Code generated by protoc-gen-go.
// source: db/version/version.proto
// DO NOT EDIT!

/*
Package version is a generated protocol buffer package.

It is generated from these files:
	db/version/version.proto

It has these top-level messages:
	V
*/
package version

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type V struct {
	Hash []byte `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`
}

func (m *V) Reset()                    { *m = V{} }
func (m *V) String() string            { return proto.CompactTextString(m) }
func (*V) ProtoMessage()               {}
func (*V) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *V) GetHash() []byte {
	if m != nil {
		return m.Hash
	}
	return nil
}

func init() {
	proto.RegisterType((*V)(nil), "version.V")
}

func init() { proto.RegisterFile("db/version/version.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 74 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xe2, 0x92, 0x48, 0x49, 0xd2, 0x2f,
	0x4b, 0x2d, 0x2a, 0xce, 0xcc, 0xcf, 0x83, 0xd1, 0x7a, 0x05, 0x45, 0xf9, 0x25, 0xf9, 0x42, 0xec,
	0x50, 0xae, 0x92, 0x38, 0x17, 0x63, 0x98, 0x90, 0x10, 0x17, 0x4b, 0x46, 0x62, 0x71, 0x86, 0x04,
	0xa3, 0x02, 0xa3, 0x06, 0x4f, 0x10, 0x98, 0x9d, 0xc4, 0x06, 0x56, 0x68, 0x0c, 0x08, 0x00, 0x00,
	0xff, 0xff, 0x5c, 0xa1, 0x2d, 0xe9, 0x44, 0x00, 0x00, 0x00,
}
