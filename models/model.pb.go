// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.0
// 	protoc        v3.21.3
// source: model.proto

package models

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// 错误定义
type Error struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Message   string `protobuf:"bytes,1,opt,name=message,proto3" json:"message,omitempty"`
	ErrorCode int64  `protobuf:"varint,2,opt,name=ErrorCode,proto3" json:"ErrorCode,omitempty"`
}

func (x *Error) Reset() {
	*x = Error{}
	if protoimpl.UnsafeEnabled {
		mi := &file_model_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Error) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Error) ProtoMessage() {}

func (x *Error) ProtoReflect() protoreflect.Message {
	mi := &file_model_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Error.ProtoReflect.Descriptor instead.
func (*Error) Descriptor() ([]byte, []int) {
	return file_model_proto_rawDescGZIP(), []int{0}
}

func (x *Error) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

func (x *Error) GetErrorCode() int64 {
	if x != nil {
		return x.ErrorCode
	}
	return 0
}

// 分页查询
type PageQuery struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	LastId      int64 `protobuf:"varint,1,opt,name=lastId,proto3" json:"lastId,omitempty"`         //最后的ID
	PageNumber  int64 `protobuf:"varint,3,opt,name=pageNumber,proto3" json:"pageNumber,omitempty"` //页数
	PageSize    int64 `protobuf:"varint,2,opt,name=pageSize,proto3" json:"pageSize,omitempty"`     //页大小
	ReturnTotal bool  `protobuf:"varint,4,opt,name=return_total,json=returnTotal,proto3" json:"return_total,omitempty"`
}

func (x *PageQuery) Reset() {
	*x = PageQuery{}
	if protoimpl.UnsafeEnabled {
		mi := &file_model_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PageQuery) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PageQuery) ProtoMessage() {}

func (x *PageQuery) ProtoReflect() protoreflect.Message {
	mi := &file_model_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PageQuery.ProtoReflect.Descriptor instead.
func (*PageQuery) Descriptor() ([]byte, []int) {
	return file_model_proto_rawDescGZIP(), []int{1}
}

func (x *PageQuery) GetLastId() int64 {
	if x != nil {
		return x.LastId
	}
	return 0
}

func (x *PageQuery) GetPageNumber() int64 {
	if x != nil {
		return x.PageNumber
	}
	return 0
}

func (x *PageQuery) GetPageSize() int64 {
	if x != nil {
		return x.PageSize
	}
	return 0
}

func (x *PageQuery) GetReturnTotal() bool {
	if x != nil {
		return x.ReturnTotal
	}
	return false
}

type PayloadRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Payload []byte `protobuf:"bytes,1,opt,name=payload,proto3" json:"payload,omitempty"`
}

func (x *PayloadRequest) Reset() {
	*x = PayloadRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_model_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PayloadRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PayloadRequest) ProtoMessage() {}

func (x *PayloadRequest) ProtoReflect() protoreflect.Message {
	mi := &file_model_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PayloadRequest.ProtoReflect.Descriptor instead.
func (*PayloadRequest) Descriptor() ([]byte, []int) {
	return file_model_proto_rawDescGZIP(), []int{2}
}

func (x *PayloadRequest) GetPayload() []byte {
	if x != nil {
		return x.Payload
	}
	return nil
}

// 通用Respose
type PayloadResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Err     *Error `protobuf:"bytes,1,opt,name=err,proto3" json:"err,omitempty"`
	Payload []byte `protobuf:"bytes,2,opt,name=payload,proto3" json:"payload,omitempty"`
}

func (x *PayloadResponse) Reset() {
	*x = PayloadResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_model_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PayloadResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PayloadResponse) ProtoMessage() {}

func (x *PayloadResponse) ProtoReflect() protoreflect.Message {
	mi := &file_model_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PayloadResponse.ProtoReflect.Descriptor instead.
func (*PayloadResponse) Descriptor() ([]byte, []int) {
	return file_model_proto_rawDescGZIP(), []int{3}
}

func (x *PayloadResponse) GetErr() *Error {
	if x != nil {
		return x.Err
	}
	return nil
}

func (x *PayloadResponse) GetPayload() []byte {
	if x != nil {
		return x.Payload
	}
	return nil
}

type CacheSingle struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Data []byte `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"` //数据
	Exp  int64  `protobuf:"varint,2,opt,name=Exp,proto3" json:"Exp,omitempty"`  //过期时间
}

func (x *CacheSingle) Reset() {
	*x = CacheSingle{}
	if protoimpl.UnsafeEnabled {
		mi := &file_model_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CacheSingle) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CacheSingle) ProtoMessage() {}

func (x *CacheSingle) ProtoReflect() protoreflect.Message {
	mi := &file_model_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CacheSingle.ProtoReflect.Descriptor instead.
func (*CacheSingle) Descriptor() ([]byte, []int) {
	return file_model_proto_rawDescGZIP(), []int{4}
}

func (x *CacheSingle) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

func (x *CacheSingle) GetExp() int64 {
	if x != nil {
		return x.Exp
	}
	return 0
}

type CacheRepeated struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Datas [][]byte `protobuf:"bytes,1,rep,name=datas,proto3" json:"datas,omitempty"` //数据
	Exp   int64    `protobuf:"varint,2,opt,name=Exp,proto3" json:"Exp,omitempty"`    //过期时间
}

func (x *CacheRepeated) Reset() {
	*x = CacheRepeated{}
	if protoimpl.UnsafeEnabled {
		mi := &file_model_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CacheRepeated) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CacheRepeated) ProtoMessage() {}

func (x *CacheRepeated) ProtoReflect() protoreflect.Message {
	mi := &file_model_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CacheRepeated.ProtoReflect.Descriptor instead.
func (*CacheRepeated) Descriptor() ([]byte, []int) {
	return file_model_proto_rawDescGZIP(), []int{5}
}

func (x *CacheRepeated) GetDatas() [][]byte {
	if x != nil {
		return x.Datas
	}
	return nil
}

func (x *CacheRepeated) GetExp() int64 {
	if x != nil {
		return x.Exp
	}
	return 0
}

type Empty struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *Empty) Reset() {
	*x = Empty{}
	if protoimpl.UnsafeEnabled {
		mi := &file_model_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Empty) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Empty) ProtoMessage() {}

func (x *Empty) ProtoReflect() protoreflect.Message {
	mi := &file_model_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Empty.ProtoReflect.Descriptor instead.
func (*Empty) Descriptor() ([]byte, []int) {
	return file_model_proto_rawDescGZIP(), []int{6}
}

var File_model_proto protoreflect.FileDescriptor

var file_model_proto_rawDesc = []byte{
	0x0a, 0x0b, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0d, 0x63,
	0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x73, 0x2e, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x22, 0x3f, 0x0a, 0x05,
	0x45, 0x72, 0x72, 0x6f, 0x72, 0x12, 0x18, 0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12,
	0x1c, 0x0a, 0x09, 0x45, 0x72, 0x72, 0x6f, 0x72, 0x43, 0x6f, 0x64, 0x65, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x03, 0x52, 0x09, 0x45, 0x72, 0x72, 0x6f, 0x72, 0x43, 0x6f, 0x64, 0x65, 0x22, 0x82, 0x01,
	0x0a, 0x09, 0x50, 0x61, 0x67, 0x65, 0x51, 0x75, 0x65, 0x72, 0x79, 0x12, 0x16, 0x0a, 0x06, 0x6c,
	0x61, 0x73, 0x74, 0x49, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x06, 0x6c, 0x61, 0x73,
	0x74, 0x49, 0x64, 0x12, 0x1e, 0x0a, 0x0a, 0x70, 0x61, 0x67, 0x65, 0x4e, 0x75, 0x6d, 0x62, 0x65,
	0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x52, 0x0a, 0x70, 0x61, 0x67, 0x65, 0x4e, 0x75, 0x6d,
	0x62, 0x65, 0x72, 0x12, 0x1a, 0x0a, 0x08, 0x70, 0x61, 0x67, 0x65, 0x53, 0x69, 0x7a, 0x65, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x08, 0x70, 0x61, 0x67, 0x65, 0x53, 0x69, 0x7a, 0x65, 0x12,
	0x21, 0x0a, 0x0c, 0x72, 0x65, 0x74, 0x75, 0x72, 0x6e, 0x5f, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x18,
	0x04, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0b, 0x72, 0x65, 0x74, 0x75, 0x72, 0x6e, 0x54, 0x6f, 0x74,
	0x61, 0x6c, 0x22, 0x2a, 0x0a, 0x0e, 0x50, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x12, 0x18, 0x0a, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x22, 0x53,
	0x0a, 0x0f, 0x50, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x12, 0x26, 0x0a, 0x03, 0x65, 0x72, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x14,
	0x2e, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x73, 0x2e, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x2e, 0x45,
	0x72, 0x72, 0x6f, 0x72, 0x52, 0x03, 0x65, 0x72, 0x72, 0x12, 0x18, 0x0a, 0x07, 0x70, 0x61, 0x79,
	0x6c, 0x6f, 0x61, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x70, 0x61, 0x79, 0x6c,
	0x6f, 0x61, 0x64, 0x22, 0x33, 0x0a, 0x0b, 0x43, 0x61, 0x63, 0x68, 0x65, 0x53, 0x69, 0x6e, 0x67,
	0x6c, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c,
	0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x12, 0x10, 0x0a, 0x03, 0x45, 0x78, 0x70, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x03, 0x52, 0x03, 0x45, 0x78, 0x70, 0x22, 0x37, 0x0a, 0x0d, 0x43, 0x61, 0x63, 0x68,
	0x65, 0x52, 0x65, 0x70, 0x65, 0x61, 0x74, 0x65, 0x64, 0x12, 0x14, 0x0a, 0x05, 0x64, 0x61, 0x74,
	0x61, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0c, 0x52, 0x05, 0x64, 0x61, 0x74, 0x61, 0x73, 0x12,
	0x10, 0x0a, 0x03, 0x45, 0x78, 0x70, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x03, 0x45, 0x78,
	0x70, 0x22, 0x07, 0x0a, 0x05, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x42, 0x2b, 0x5a, 0x29, 0x67, 0x69,
	0x74, 0x75, 0x68, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6f, 0x66, 0x66, 0x65, 0x65, 0x68,
	0x63, 0x2f, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x73, 0x2f, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x73,
	0x3b, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_model_proto_rawDescOnce sync.Once
	file_model_proto_rawDescData = file_model_proto_rawDesc
)

func file_model_proto_rawDescGZIP() []byte {
	file_model_proto_rawDescOnce.Do(func() {
		file_model_proto_rawDescData = protoimpl.X.CompressGZIP(file_model_proto_rawDescData)
	})
	return file_model_proto_rawDescData
}

var file_model_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_model_proto_goTypes = []interface{}{
	(*Error)(nil),           // 0: commons.model.Error
	(*PageQuery)(nil),       // 1: commons.model.PageQuery
	(*PayloadRequest)(nil),  // 2: commons.model.PayloadRequest
	(*PayloadResponse)(nil), // 3: commons.model.PayloadResponse
	(*CacheSingle)(nil),     // 4: commons.model.CacheSingle
	(*CacheRepeated)(nil),   // 5: commons.model.CacheRepeated
	(*Empty)(nil),           // 6: commons.model.Empty
}
var file_model_proto_depIdxs = []int32{
	0, // 0: commons.model.PayloadResponse.err:type_name -> commons.model.Error
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_model_proto_init() }
func file_model_proto_init() {
	if File_model_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_model_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Error); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_model_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PageQuery); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_model_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PayloadRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_model_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PayloadResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_model_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CacheSingle); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_model_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CacheRepeated); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_model_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Empty); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_model_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_model_proto_goTypes,
		DependencyIndexes: file_model_proto_depIdxs,
		MessageInfos:      file_model_proto_msgTypes,
	}.Build()
	File_model_proto = out.File
	file_model_proto_rawDesc = nil
	file_model_proto_goTypes = nil
	file_model_proto_depIdxs = nil
}
