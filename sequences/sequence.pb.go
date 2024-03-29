// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: sequence.proto

package sequences

import (
	fmt "fmt"
	proto "github.com/gogo/protobuf/proto"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

type Sequence struct {
	Id                   int64    `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	CreateTimeMS         int64    `protobuf:"varint,2,opt,name=createTimeMS,proto3" json:"createTimeMS,omitempty"`
	NodeID               int64    `protobuf:"varint,3,opt,name=nodeID,proto3" json:"nodeID,omitempty"`
	Index                int64    `protobuf:"varint,4,opt,name=index,proto3" json:"index,omitempty"`
	DcID                 int64    `protobuf:"varint,5,opt,name=dcID,proto3" json:"dcID,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Sequence) Reset()         { *m = Sequence{} }
func (m *Sequence) String() string { return proto.CompactTextString(m) }
func (*Sequence) ProtoMessage()    {}
func (*Sequence) Descriptor() ([]byte, []int) {
	return fileDescriptor_e97b888ecada2421, []int{0}
}
func (m *Sequence) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Sequence.Unmarshal(m, b)
}
func (m *Sequence) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Sequence.Marshal(b, m, deterministic)
}
func (m *Sequence) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Sequence.Merge(m, src)
}
func (m *Sequence) XXX_Size() int {
	return xxx_messageInfo_Sequence.Size(m)
}
func (m *Sequence) XXX_DiscardUnknown() {
	xxx_messageInfo_Sequence.DiscardUnknown(m)
}

var xxx_messageInfo_Sequence proto.InternalMessageInfo

func (m *Sequence) GetId() int64 {
	if m != nil {
		return m.Id
	}
	return 0
}

func (m *Sequence) GetCreateTimeMS() int64 {
	if m != nil {
		return m.CreateTimeMS
	}
	return 0
}

func (m *Sequence) GetNodeID() int64 {
	if m != nil {
		return m.NodeID
	}
	return 0
}

func (m *Sequence) GetIndex() int64 {
	if m != nil {
		return m.Index
	}
	return 0
}

func (m *Sequence) GetDcID() int64 {
	if m != nil {
		return m.DcID
	}
	return 0
}

func init() {
	proto.RegisterType((*Sequence)(nil), "commons.sequences.Sequence")
}

func init() { proto.RegisterFile("sequence.proto", fileDescriptor_e97b888ecada2421) }

var fileDescriptor_e97b888ecada2421 = []byte{
	// 154 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x2b, 0x4e, 0x2d, 0x2c,
	0x4d, 0xcd, 0x4b, 0x4e, 0xd5, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x12, 0x4c, 0xce, 0xcf, 0xcd,
	0xcd, 0xcf, 0x2b, 0xd6, 0x83, 0x89, 0x17, 0x2b, 0x35, 0x30, 0x72, 0x71, 0x04, 0x43, 0x79, 0x42,
	0x7c, 0x5c, 0x4c, 0x99, 0x29, 0x12, 0x8c, 0x0a, 0x8c, 0x1a, 0xcc, 0x41, 0x4c, 0x99, 0x29, 0x42,
	0x4a, 0x5c, 0x3c, 0xc9, 0x45, 0xa9, 0x89, 0x25, 0xa9, 0x21, 0x99, 0xb9, 0xa9, 0xbe, 0xc1, 0x12,
	0x4c, 0x60, 0x19, 0x14, 0x31, 0x21, 0x31, 0x2e, 0xb6, 0xbc, 0xfc, 0x94, 0x54, 0x4f, 0x17, 0x09,
	0x66, 0xb0, 0x2c, 0x94, 0x27, 0x24, 0xc2, 0xc5, 0x9a, 0x99, 0x97, 0x92, 0x5a, 0x21, 0xc1, 0x02,
	0x16, 0x86, 0x70, 0x84, 0x84, 0xb8, 0x58, 0x52, 0x92, 0x3d, 0x5d, 0x24, 0x58, 0xc1, 0x82, 0x60,
	0xb6, 0x13, 0x77, 0x14, 0x27, 0xdc, 0x3d, 0x49, 0x6c, 0x60, 0x97, 0x1a, 0x03, 0x02, 0x00, 0x00,
	0xff, 0xff, 0x01, 0xa6, 0x58, 0x62, 0xbb, 0x00, 0x00, 0x00,
}
