// Code generated by protoc-gen-go. DO NOT EDIT.
// source: myc/protocol/gossip.proto

package protocol

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import version "gitlab.com/SporeDB/sporedb/db/version"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type Hello struct {
	Version  uint64 `protobuf:"varint,1,opt,name=version" json:"version,omitempty"`
	Identity string `protobuf:"bytes,2,opt,name=identity" json:"identity,omitempty"`
}

func (m *Hello) Reset()                    { *m = Hello{} }
func (m *Hello) String() string            { return proto.CompactTextString(m) }
func (*Hello) ProtoMessage()               {}
func (*Hello) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *Hello) GetVersion() uint64 {
	if m != nil {
		return m.Version
	}
	return 0
}

func (m *Hello) GetIdentity() string {
	if m != nil {
		return m.Identity
	}
	return ""
}

type Raw struct {
	Key       string     `protobuf:"bytes,1,opt,name=key" json:"key,omitempty"`
	Version   *version.V `protobuf:"bytes,2,opt,name=version" json:"version,omitempty"`
	Data      []byte     `protobuf:"bytes,3,opt,name=data,proto3" json:"data,omitempty"`
	Signature []byte     `protobuf:"bytes,10,opt,name=signature,proto3" json:"signature,omitempty"`
}

func (m *Raw) Reset()                    { *m = Raw{} }
func (m *Raw) String() string            { return proto.CompactTextString(m) }
func (*Raw) ProtoMessage()               {}
func (*Raw) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *Raw) GetKey() string {
	if m != nil {
		return m.Key
	}
	return ""
}

func (m *Raw) GetVersion() *version.V {
	if m != nil {
		return m.Version
	}
	return nil
}

func (m *Raw) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

func (m *Raw) GetSignature() []byte {
	if m != nil {
		return m.Signature
	}
	return nil
}

type Node struct {
	Identity string `protobuf:"bytes,1,opt,name=identity" json:"identity,omitempty"`
	Address  string `protobuf:"bytes,2,opt,name=address" json:"address,omitempty"`
}

func (m *Node) Reset()                    { *m = Node{} }
func (m *Node) String() string            { return proto.CompactTextString(m) }
func (*Node) ProtoMessage()               {}
func (*Node) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *Node) GetIdentity() string {
	if m != nil {
		return m.Identity
	}
	return ""
}

func (m *Node) GetAddress() string {
	if m != nil {
		return m.Address
	}
	return ""
}

type Nodes struct {
	Nodes []*Node `protobuf:"bytes,1,rep,name=nodes" json:"nodes,omitempty"`
}

func (m *Nodes) Reset()                    { *m = Nodes{} }
func (m *Nodes) String() string            { return proto.CompactTextString(m) }
func (*Nodes) ProtoMessage()               {}
func (*Nodes) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func (m *Nodes) GetNodes() []*Node {
	if m != nil {
		return m.Nodes
	}
	return nil
}

type GossipUnit struct {
	SporeUuid string `protobuf:"bytes,1,opt,name=spore_uuid,json=sporeUuid" json:"spore_uuid,omitempty"`
	Endorser  string `protobuf:"bytes,2,opt,name=endorser" json:"endorser,omitempty"`
}

func (m *GossipUnit) Reset()                    { *m = GossipUnit{} }
func (m *GossipUnit) String() string            { return proto.CompactTextString(m) }
func (*GossipUnit) ProtoMessage()               {}
func (*GossipUnit) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

func (m *GossipUnit) GetSporeUuid() string {
	if m != nil {
		return m.SporeUuid
	}
	return ""
}

func (m *GossipUnit) GetEndorser() string {
	if m != nil {
		return m.Endorser
	}
	return ""
}

type Gossip struct {
	Units []*GossipUnit `protobuf:"bytes,1,rep,name=units" json:"units,omitempty"`
}

func (m *Gossip) Reset()                    { *m = Gossip{} }
func (m *Gossip) String() string            { return proto.CompactTextString(m) }
func (*Gossip) ProtoMessage()               {}
func (*Gossip) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{5} }

func (m *Gossip) GetUnits() []*GossipUnit {
	if m != nil {
		return m.Units
	}
	return nil
}

func init() {
	proto.RegisterType((*Hello)(nil), "protocol.Hello")
	proto.RegisterType((*Raw)(nil), "protocol.Raw")
	proto.RegisterType((*Node)(nil), "protocol.Node")
	proto.RegisterType((*Nodes)(nil), "protocol.Nodes")
	proto.RegisterType((*GossipUnit)(nil), "protocol.GossipUnit")
	proto.RegisterType((*Gossip)(nil), "protocol.Gossip")
}

func init() { proto.RegisterFile("myc/protocol/gossip.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 294 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x54, 0x50, 0x41, 0x4b, 0xf4, 0x30,
	0x10, 0x25, 0x5f, 0xdb, 0xfd, 0xb6, 0xb3, 0x22, 0x12, 0x3c, 0xc4, 0x45, 0xa1, 0x84, 0x3d, 0x14,
	0xc1, 0x16, 0x56, 0x8f, 0x7a, 0x5e, 0x4f, 0x1e, 0x02, 0xeb, 0x55, 0xba, 0x9b, 0xb0, 0x04, 0x6b,
	0x52, 0x92, 0x54, 0xe9, 0xbf, 0x97, 0xa4, 0x8d, 0x65, 0x4f, 0x99, 0x37, 0x6f, 0x66, 0xde, 0xcb,
	0x83, 0x9b, 0xaf, 0xe1, 0x58, 0x77, 0x46, 0x3b, 0x7d, 0xd4, 0x6d, 0x7d, 0xd2, 0xd6, 0xca, 0xae,
	0x0a, 0x18, 0x2f, 0x63, 0x7b, 0x4d, 0xf8, 0xa1, 0xfe, 0x16, 0xc6, 0x4a, 0xad, 0xe2, 0x3b, 0xce,
	0xd0, 0x17, 0xc8, 0x5e, 0x45, 0xdb, 0x6a, 0x4c, 0xe0, 0xff, 0xc4, 0x10, 0x54, 0xa0, 0x32, 0x65,
	0x11, 0xe2, 0x35, 0x2c, 0x25, 0x17, 0xca, 0x49, 0x37, 0x90, 0x7f, 0x05, 0x2a, 0x73, 0xf6, 0x87,
	0xa9, 0x86, 0x84, 0x35, 0x3f, 0xf8, 0x0a, 0x92, 0x4f, 0x31, 0x84, 0xc5, 0x9c, 0xf9, 0x12, 0x6f,
	0xe6, 0x73, 0x7e, 0x67, 0xb5, 0x85, 0x2a, 0x0a, 0xbf, 0xcf, 0xa7, 0x31, 0xa4, 0xbc, 0x71, 0x0d,
	0x49, 0x0a, 0x54, 0x5e, 0xb0, 0x50, 0xe3, 0x5b, 0xc8, 0xad, 0x3c, 0xa9, 0xc6, 0xf5, 0x46, 0x10,
	0x08, 0xc4, 0xdc, 0xa0, 0xcf, 0x90, 0xbe, 0x69, 0x2e, 0xce, 0x4c, 0xa1, 0x73, 0x53, 0xfe, 0x2b,
	0x0d, 0xe7, 0x46, 0x58, 0x3b, 0xf9, 0x8d, 0x90, 0x3e, 0x40, 0xe6, 0xb7, 0x2d, 0xde, 0x40, 0xa6,
	0x7c, 0x41, 0x50, 0x91, 0x94, 0xab, 0xed, 0x65, 0x15, 0xa3, 0xaa, 0x3c, 0xcf, 0x46, 0x92, 0xee,
	0x00, 0x76, 0x21, 0xd0, 0xbd, 0x92, 0x0e, 0xdf, 0x01, 0xd8, 0x4e, 0x1b, 0xf1, 0xd1, 0xf7, 0x92,
	0x4f, 0xa2, 0x79, 0xe8, 0xec, 0x7b, 0xc9, 0xbd, 0x23, 0xa1, 0xb8, 0x36, 0x56, 0x98, 0x18, 0x53,
	0xc4, 0xf4, 0x09, 0x16, 0xe3, 0x21, 0x7c, 0x0f, 0x59, 0xaf, 0xa4, 0x8b, 0xc2, 0xd7, 0xb3, 0xf0,
	0xac, 0xc4, 0xc6, 0x91, 0xc3, 0x22, 0x70, 0x8f, 0xbf, 0x01, 0x00, 0x00, 0xff, 0xff, 0x02, 0x12,
	0x87, 0x2d, 0xe3, 0x01, 0x00, 0x00,
}
