// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: builder.proto

package sqlbuilder

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

type ValueType int32

const (
	ValueType_ValueType_    ValueType = 0
	ValueType_Int           ValueType = 1
	ValueType_Doubel        ValueType = 2
	ValueType_String        ValueType = 3
	ValueType_Bool          ValueType = 4
	ValueType_Time          ValueType = 5
	ValueType_StringArray   ValueType = 6
	ValueType_IntArray      ValueType = 7
	ValueType_DoubleArray   ValueType = 8
	ValueType_TimeArray     ValueType = 9
	ValueType_Status        ValueType = 10
	ValueType_Statuses      ValueType = 11
	ValueType_PgIntArray    ValueType = 12
	ValueType_PgStringArray ValueType = 13
	ValueType_PgFloatArray  ValueType = 14
)

var ValueType_name = map[int32]string{
	0:  "ValueType_",
	1:  "Int",
	2:  "Doubel",
	3:  "String",
	4:  "Bool",
	5:  "Time",
	6:  "StringArray",
	7:  "IntArray",
	8:  "DoubleArray",
	9:  "TimeArray",
	10: "Status",
	11: "Statuses",
	12: "PgIntArray",
	13: "PgStringArray",
	14: "PgFloatArray",
}

var ValueType_value = map[string]int32{
	"ValueType_":    0,
	"Int":           1,
	"Doubel":        2,
	"String":        3,
	"Bool":          4,
	"Time":          5,
	"StringArray":   6,
	"IntArray":      7,
	"DoubleArray":   8,
	"TimeArray":     9,
	"Status":        10,
	"Statuses":      11,
	"PgIntArray":    12,
	"PgStringArray": 13,
	"PgFloatArray":  14,
}

func (x ValueType) String() string {
	return proto.EnumName(ValueType_name, int32(x))
}

func (ValueType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_68a5e6cb4f7c8dc9, []int{0}
}

type RangeType int32

const (
	RangeType_ALL     RangeType = 0
	RangeType_Minute  RangeType = 1
	RangeType_MinuteN RangeType = 2
	RangeType_Hour    RangeType = 3
	RangeType_Day     RangeType = 4
	RangeType_Week    RangeType = 5
	RangeType_Month   RangeType = 6
	RangeType_Year    RangeType = 7
)

var RangeType_name = map[int32]string{
	0: "ALL",
	1: "Minute",
	2: "MinuteN",
	3: "Hour",
	4: "Day",
	5: "Week",
	6: "Month",
	7: "Year",
}

var RangeType_value = map[string]int32{
	"ALL":     0,
	"Minute":  1,
	"MinuteN": 2,
	"Hour":    3,
	"Day":     4,
	"Week":    5,
	"Month":   6,
	"Year":    7,
}

func (x RangeType) String() string {
	return proto.EnumName(RangeType_name, int32(x))
}

func (RangeType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_68a5e6cb4f7c8dc9, []int{1}
}

type TaskStatus int32

const (
	TaskStatus_TaskStatus_ TaskStatus = 0
	TaskStatus_Running     TaskStatus = 1
	TaskStatus_Success     TaskStatus = 2
	TaskStatus_Error       TaskStatus = 3
)

var TaskStatus_name = map[int32]string{
	0: "TaskStatus_",
	1: "Running",
	2: "Success",
	3: "Error",
}

var TaskStatus_value = map[string]int32{
	"TaskStatus_": 0,
	"Running":     1,
	"Success":     2,
	"Error":       3,
}

func (x TaskStatus) String() string {
	return proto.EnumName(TaskStatus_name, int32(x))
}

func (TaskStatus) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_68a5e6cb4f7c8dc9, []int{2}
}

type PageQuery struct {
	PageSize             int64    `protobuf:"varint,1,opt,name=page_size,json=pageSize,proto3" json:"page_size,omitempty"`
	PageIndex            int64    `protobuf:"varint,2,opt,name=page_index,json=pageIndex,proto3" json:"page_index,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *PageQuery) Reset()         { *m = PageQuery{} }
func (m *PageQuery) String() string { return proto.CompactTextString(m) }
func (*PageQuery) ProtoMessage()    {}
func (*PageQuery) Descriptor() ([]byte, []int) {
	return fileDescriptor_68a5e6cb4f7c8dc9, []int{0}
}
func (m *PageQuery) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PageQuery.Unmarshal(m, b)
}
func (m *PageQuery) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PageQuery.Marshal(b, m, deterministic)
}
func (m *PageQuery) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PageQuery.Merge(m, src)
}
func (m *PageQuery) XXX_Size() int {
	return xxx_messageInfo_PageQuery.Size(m)
}
func (m *PageQuery) XXX_DiscardUnknown() {
	xxx_messageInfo_PageQuery.DiscardUnknown(m)
}

var xxx_messageInfo_PageQuery proto.InternalMessageInfo

func (m *PageQuery) GetPageSize() int64 {
	if m != nil {
		return m.PageSize
	}
	return 0
}

func (m *PageQuery) GetPageIndex() int64 {
	if m != nil {
		return m.PageIndex
	}
	return 0
}

type Value struct {
	ValueType            ValueType `protobuf:"varint,1,opt,name=value_type,json=valueType,proto3,enum=commons.sqlbuilder.ValueType" json:"value_type,omitempty"`
	IntValue             int64     `protobuf:"varint,2,opt,name=int_value,json=intValue,proto3" json:"int_value,omitempty"`
	DoubleValue          float64   `protobuf:"fixed64,3,opt,name=double_value,json=doubleValue,proto3" json:"double_value,omitempty"`
	StringValue          string    `protobuf:"bytes,4,opt,name=string_value,json=stringValue,proto3" json:"string_value,omitempty"`
	BoolValue            bool      `protobuf:"varint,5,opt,name=bool_value,json=boolValue,proto3" json:"bool_value,omitempty"`
	IntValues            []int64   `protobuf:"varint,6,rep,packed,name=int_values,json=intValues,proto3" json:"int_values,omitempty"`
	StringValues         []string  `protobuf:"bytes,8,rep,name=string_values,json=stringValues,proto3" json:"string_values,omitempty"`
	DoubleValues         []float64 `protobuf:"fixed64,9,rep,packed,name=double_values,json=doubleValues,proto3" json:"double_values,omitempty"`
	XXX_NoUnkeyedLiteral struct{}  `json:"-"`
	XXX_unrecognized     []byte    `json:"-"`
	XXX_sizecache        int32     `json:"-"`
}

func (m *Value) Reset()         { *m = Value{} }
func (m *Value) String() string { return proto.CompactTextString(m) }
func (*Value) ProtoMessage()    {}
func (*Value) Descriptor() ([]byte, []int) {
	return fileDescriptor_68a5e6cb4f7c8dc9, []int{1}
}
func (m *Value) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Value.Unmarshal(m, b)
}
func (m *Value) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Value.Marshal(b, m, deterministic)
}
func (m *Value) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Value.Merge(m, src)
}
func (m *Value) XXX_Size() int {
	return xxx_messageInfo_Value.Size(m)
}
func (m *Value) XXX_DiscardUnknown() {
	xxx_messageInfo_Value.DiscardUnknown(m)
}

var xxx_messageInfo_Value proto.InternalMessageInfo

func (m *Value) GetValueType() ValueType {
	if m != nil {
		return m.ValueType
	}
	return ValueType_ValueType_
}

func (m *Value) GetIntValue() int64 {
	if m != nil {
		return m.IntValue
	}
	return 0
}

func (m *Value) GetDoubleValue() float64 {
	if m != nil {
		return m.DoubleValue
	}
	return 0
}

func (m *Value) GetStringValue() string {
	if m != nil {
		return m.StringValue
	}
	return ""
}

func (m *Value) GetBoolValue() bool {
	if m != nil {
		return m.BoolValue
	}
	return false
}

func (m *Value) GetIntValues() []int64 {
	if m != nil {
		return m.IntValues
	}
	return nil
}

func (m *Value) GetStringValues() []string {
	if m != nil {
		return m.StringValues
	}
	return nil
}

func (m *Value) GetDoubleValues() []float64 {
	if m != nil {
		return m.DoubleValues
	}
	return nil
}

type JoinCondition struct {
	TableName            string   `protobuf:"bytes,1,opt,name=table_name,json=tableName,proto3" json:"table_name,omitempty"`
	TableColName         string   `protobuf:"bytes,2,opt,name=table_col_name,json=tableColName,proto3" json:"table_col_name,omitempty"`
	JoinTableColName     string   `protobuf:"bytes,3,opt,name=join_table_col_name,json=joinTableColName,proto3" json:"join_table_col_name,omitempty"`
	TableAlisa           string   `protobuf:"bytes,4,opt,name=table_alisa,json=tableAlisa,proto3" json:"table_alisa,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *JoinCondition) Reset()         { *m = JoinCondition{} }
func (m *JoinCondition) String() string { return proto.CompactTextString(m) }
func (*JoinCondition) ProtoMessage()    {}
func (*JoinCondition) Descriptor() ([]byte, []int) {
	return fileDescriptor_68a5e6cb4f7c8dc9, []int{2}
}
func (m *JoinCondition) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_JoinCondition.Unmarshal(m, b)
}
func (m *JoinCondition) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_JoinCondition.Marshal(b, m, deterministic)
}
func (m *JoinCondition) XXX_Merge(src proto.Message) {
	xxx_messageInfo_JoinCondition.Merge(m, src)
}
func (m *JoinCondition) XXX_Size() int {
	return xxx_messageInfo_JoinCondition.Size(m)
}
func (m *JoinCondition) XXX_DiscardUnknown() {
	xxx_messageInfo_JoinCondition.DiscardUnknown(m)
}

var xxx_messageInfo_JoinCondition proto.InternalMessageInfo

func (m *JoinCondition) GetTableName() string {
	if m != nil {
		return m.TableName
	}
	return ""
}

func (m *JoinCondition) GetTableColName() string {
	if m != nil {
		return m.TableColName
	}
	return ""
}

func (m *JoinCondition) GetJoinTableColName() string {
	if m != nil {
		return m.JoinTableColName
	}
	return ""
}

func (m *JoinCondition) GetTableAlisa() string {
	if m != nil {
		return m.TableAlisa
	}
	return ""
}

type Condition struct {
	ColName              string   `protobuf:"bytes,1,opt,name=col_name,json=colName,proto3" json:"col_name,omitempty"`
	Operator             string   `protobuf:"bytes,2,opt,name=operator,proto3" json:"operator,omitempty"`
	Value                *Value   `protobuf:"bytes,3,opt,name=value,proto3" json:"value,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Condition) Reset()         { *m = Condition{} }
func (m *Condition) String() string { return proto.CompactTextString(m) }
func (*Condition) ProtoMessage()    {}
func (*Condition) Descriptor() ([]byte, []int) {
	return fileDescriptor_68a5e6cb4f7c8dc9, []int{3}
}
func (m *Condition) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Condition.Unmarshal(m, b)
}
func (m *Condition) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Condition.Marshal(b, m, deterministic)
}
func (m *Condition) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Condition.Merge(m, src)
}
func (m *Condition) XXX_Size() int {
	return xxx_messageInfo_Condition.Size(m)
}
func (m *Condition) XXX_DiscardUnknown() {
	xxx_messageInfo_Condition.DiscardUnknown(m)
}

var xxx_messageInfo_Condition proto.InternalMessageInfo

func (m *Condition) GetColName() string {
	if m != nil {
		return m.ColName
	}
	return ""
}

func (m *Condition) GetOperator() string {
	if m != nil {
		return m.Operator
	}
	return ""
}

func (m *Condition) GetValue() *Value {
	if m != nil {
		return m.Value
	}
	return nil
}

type OrderCondition struct {
	Name                 string   `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Order                string   `protobuf:"bytes,2,opt,name=order,proto3" json:"order,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *OrderCondition) Reset()         { *m = OrderCondition{} }
func (m *OrderCondition) String() string { return proto.CompactTextString(m) }
func (*OrderCondition) ProtoMessage()    {}
func (*OrderCondition) Descriptor() ([]byte, []int) {
	return fileDescriptor_68a5e6cb4f7c8dc9, []int{4}
}
func (m *OrderCondition) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_OrderCondition.Unmarshal(m, b)
}
func (m *OrderCondition) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_OrderCondition.Marshal(b, m, deterministic)
}
func (m *OrderCondition) XXX_Merge(src proto.Message) {
	xxx_messageInfo_OrderCondition.Merge(m, src)
}
func (m *OrderCondition) XXX_Size() int {
	return xxx_messageInfo_OrderCondition.Size(m)
}
func (m *OrderCondition) XXX_DiscardUnknown() {
	xxx_messageInfo_OrderCondition.DiscardUnknown(m)
}

var xxx_messageInfo_OrderCondition proto.InternalMessageInfo

func (m *OrderCondition) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *OrderCondition) GetOrder() string {
	if m != nil {
		return m.Order
	}
	return ""
}

type Query struct {
	Conditions           []*Condition      `protobuf:"bytes,1,rep,name=conditions,proto3" json:"conditions,omitempty"`
	OrderConditions      []*OrderCondition `protobuf:"bytes,2,rep,name=order_conditions,json=orderConditions,proto3" json:"order_conditions,omitempty"`
	Page                 *PageQuery        `protobuf:"bytes,3,opt,name=page,proto3" json:"page,omitempty"`
	ReturnTotal          bool              `protobuf:"varint,4,opt,name=return_total,json=returnTotal,proto3" json:"return_total,omitempty"`
	ForcePage            bool              `protobuf:"varint,5,opt,name=force_page,json=forcePage,proto3" json:"force_page,omitempty"`
	Join                 *JoinCondition    `protobuf:"bytes,6,opt,name=join,proto3" json:"join,omitempty"`
	Pk                   string            `protobuf:"bytes,7,opt,name=pk,proto3" json:"pk,omitempty"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *Query) Reset()         { *m = Query{} }
func (m *Query) String() string { return proto.CompactTextString(m) }
func (*Query) ProtoMessage()    {}
func (*Query) Descriptor() ([]byte, []int) {
	return fileDescriptor_68a5e6cb4f7c8dc9, []int{5}
}
func (m *Query) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Query.Unmarshal(m, b)
}
func (m *Query) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Query.Marshal(b, m, deterministic)
}
func (m *Query) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Query.Merge(m, src)
}
func (m *Query) XXX_Size() int {
	return xxx_messageInfo_Query.Size(m)
}
func (m *Query) XXX_DiscardUnknown() {
	xxx_messageInfo_Query.DiscardUnknown(m)
}

var xxx_messageInfo_Query proto.InternalMessageInfo

func (m *Query) GetConditions() []*Condition {
	if m != nil {
		return m.Conditions
	}
	return nil
}

func (m *Query) GetOrderConditions() []*OrderCondition {
	if m != nil {
		return m.OrderConditions
	}
	return nil
}

func (m *Query) GetPage() *PageQuery {
	if m != nil {
		return m.Page
	}
	return nil
}

func (m *Query) GetReturnTotal() bool {
	if m != nil {
		return m.ReturnTotal
	}
	return false
}

func (m *Query) GetForcePage() bool {
	if m != nil {
		return m.ForcePage
	}
	return false
}

func (m *Query) GetJoin() *JoinCondition {
	if m != nil {
		return m.Join
	}
	return nil
}

func (m *Query) GetPk() string {
	if m != nil {
		return m.Pk
	}
	return ""
}

type Update struct {
	Id                   int64        `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Fields               []*Field     `protobuf:"bytes,2,rep,name=fields,proto3" json:"fields,omitempty"`
	Conditions           []*Condition `protobuf:"bytes,3,rep,name=conditions,proto3" json:"conditions,omitempty"`
	XXX_NoUnkeyedLiteral struct{}     `json:"-"`
	XXX_unrecognized     []byte       `json:"-"`
	XXX_sizecache        int32        `json:"-"`
}

func (m *Update) Reset()         { *m = Update{} }
func (m *Update) String() string { return proto.CompactTextString(m) }
func (*Update) ProtoMessage()    {}
func (*Update) Descriptor() ([]byte, []int) {
	return fileDescriptor_68a5e6cb4f7c8dc9, []int{6}
}
func (m *Update) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Update.Unmarshal(m, b)
}
func (m *Update) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Update.Marshal(b, m, deterministic)
}
func (m *Update) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Update.Merge(m, src)
}
func (m *Update) XXX_Size() int {
	return xxx_messageInfo_Update.Size(m)
}
func (m *Update) XXX_DiscardUnknown() {
	xxx_messageInfo_Update.DiscardUnknown(m)
}

var xxx_messageInfo_Update proto.InternalMessageInfo

func (m *Update) GetId() int64 {
	if m != nil {
		return m.Id
	}
	return 0
}

func (m *Update) GetFields() []*Field {
	if m != nil {
		return m.Fields
	}
	return nil
}

func (m *Update) GetConditions() []*Condition {
	if m != nil {
		return m.Conditions
	}
	return nil
}

type Field struct {
	ColName              string   `protobuf:"bytes,1,opt,name=col_name,json=colName,proto3" json:"col_name,omitempty"`
	FieldValue           *Value   `protobuf:"bytes,2,opt,name=field_value,json=fieldValue,proto3" json:"field_value,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Field) Reset()         { *m = Field{} }
func (m *Field) String() string { return proto.CompactTextString(m) }
func (*Field) ProtoMessage()    {}
func (*Field) Descriptor() ([]byte, []int) {
	return fileDescriptor_68a5e6cb4f7c8dc9, []int{7}
}
func (m *Field) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Field.Unmarshal(m, b)
}
func (m *Field) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Field.Marshal(b, m, deterministic)
}
func (m *Field) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Field.Merge(m, src)
}
func (m *Field) XXX_Size() int {
	return xxx_messageInfo_Field.Size(m)
}
func (m *Field) XXX_DiscardUnknown() {
	xxx_messageInfo_Field.DiscardUnknown(m)
}

var xxx_messageInfo_Field proto.InternalMessageInfo

func (m *Field) GetColName() string {
	if m != nil {
		return m.ColName
	}
	return ""
}

func (m *Field) GetFieldValue() *Value {
	if m != nil {
		return m.FieldValue
	}
	return nil
}

type UpdateResult struct {
	Count                int64    `protobuf:"varint,1,opt,name=count,proto3" json:"count,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *UpdateResult) Reset()         { *m = UpdateResult{} }
func (m *UpdateResult) String() string { return proto.CompactTextString(m) }
func (*UpdateResult) ProtoMessage()    {}
func (*UpdateResult) Descriptor() ([]byte, []int) {
	return fileDescriptor_68a5e6cb4f7c8dc9, []int{8}
}
func (m *UpdateResult) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_UpdateResult.Unmarshal(m, b)
}
func (m *UpdateResult) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_UpdateResult.Marshal(b, m, deterministic)
}
func (m *UpdateResult) XXX_Merge(src proto.Message) {
	xxx_messageInfo_UpdateResult.Merge(m, src)
}
func (m *UpdateResult) XXX_Size() int {
	return xxx_messageInfo_UpdateResult.Size(m)
}
func (m *UpdateResult) XXX_DiscardUnknown() {
	xxx_messageInfo_UpdateResult.DiscardUnknown(m)
}

var xxx_messageInfo_UpdateResult proto.InternalMessageInfo

func (m *UpdateResult) GetCount() int64 {
	if m != nil {
		return m.Count
	}
	return 0
}

type StatisticsTimeRange struct {
	Type                 RangeType `protobuf:"varint,1,opt,name=type,proto3,enum=commons.sqlbuilder.RangeType" json:"type,omitempty"`
	Interval             int64     `protobuf:"varint,2,opt,name=interval,proto3" json:"interval,omitempty"`
	Start                int64     `protobuf:"varint,3,opt,name=start,proto3" json:"start,omitempty"`
	End                  int64     `protobuf:"varint,4,opt,name=end,proto3" json:"end,omitempty"`
	XXX_NoUnkeyedLiteral struct{}  `json:"-"`
	XXX_unrecognized     []byte    `json:"-"`
	XXX_sizecache        int32     `json:"-"`
}

func (m *StatisticsTimeRange) Reset()         { *m = StatisticsTimeRange{} }
func (m *StatisticsTimeRange) String() string { return proto.CompactTextString(m) }
func (*StatisticsTimeRange) ProtoMessage()    {}
func (*StatisticsTimeRange) Descriptor() ([]byte, []int) {
	return fileDescriptor_68a5e6cb4f7c8dc9, []int{9}
}
func (m *StatisticsTimeRange) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_StatisticsTimeRange.Unmarshal(m, b)
}
func (m *StatisticsTimeRange) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_StatisticsTimeRange.Marshal(b, m, deterministic)
}
func (m *StatisticsTimeRange) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StatisticsTimeRange.Merge(m, src)
}
func (m *StatisticsTimeRange) XXX_Size() int {
	return xxx_messageInfo_StatisticsTimeRange.Size(m)
}
func (m *StatisticsTimeRange) XXX_DiscardUnknown() {
	xxx_messageInfo_StatisticsTimeRange.DiscardUnknown(m)
}

var xxx_messageInfo_StatisticsTimeRange proto.InternalMessageInfo

func (m *StatisticsTimeRange) GetType() RangeType {
	if m != nil {
		return m.Type
	}
	return RangeType_ALL
}

func (m *StatisticsTimeRange) GetInterval() int64 {
	if m != nil {
		return m.Interval
	}
	return 0
}

func (m *StatisticsTimeRange) GetStart() int64 {
	if m != nil {
		return m.Start
	}
	return 0
}

func (m *StatisticsTimeRange) GetEnd() int64 {
	if m != nil {
		return m.End
	}
	return 0
}

type StatisticsTask struct {
	Id                   int64      `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Task                 int64      `protobuf:"varint,2,opt,name=task,proto3" json:"task,omitempty"`
	Start                int64      `protobuf:"varint,3,opt,name=start,proto3" json:"start,omitempty"`
	End                  int64      `protobuf:"varint,4,opt,name=end,proto3" json:"end,omitempty"`
	Status               TaskStatus `protobuf:"varint,5,opt,name=status,proto3,enum=commons.sqlbuilder.TaskStatus" json:"status,omitempty"`
	Message              string     `protobuf:"bytes,6,opt,name=message,proto3" json:"message,omitempty"`
	CreateTime           int64      `protobuf:"varint,7,opt,name=create_time,json=createTime,proto3" json:"create_time,omitempty"`
	UpdateTime           int64      `protobuf:"varint,8,opt,name=update_time,json=updateTime,proto3" json:"update_time,omitempty"`
	XXX_NoUnkeyedLiteral struct{}   `json:"-"`
	XXX_unrecognized     []byte     `json:"-"`
	XXX_sizecache        int32      `json:"-"`
}

func (m *StatisticsTask) Reset()         { *m = StatisticsTask{} }
func (m *StatisticsTask) String() string { return proto.CompactTextString(m) }
func (*StatisticsTask) ProtoMessage()    {}
func (*StatisticsTask) Descriptor() ([]byte, []int) {
	return fileDescriptor_68a5e6cb4f7c8dc9, []int{10}
}
func (m *StatisticsTask) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_StatisticsTask.Unmarshal(m, b)
}
func (m *StatisticsTask) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_StatisticsTask.Marshal(b, m, deterministic)
}
func (m *StatisticsTask) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StatisticsTask.Merge(m, src)
}
func (m *StatisticsTask) XXX_Size() int {
	return xxx_messageInfo_StatisticsTask.Size(m)
}
func (m *StatisticsTask) XXX_DiscardUnknown() {
	xxx_messageInfo_StatisticsTask.DiscardUnknown(m)
}

var xxx_messageInfo_StatisticsTask proto.InternalMessageInfo

func (m *StatisticsTask) GetId() int64 {
	if m != nil {
		return m.Id
	}
	return 0
}

func (m *StatisticsTask) GetTask() int64 {
	if m != nil {
		return m.Task
	}
	return 0
}

func (m *StatisticsTask) GetStart() int64 {
	if m != nil {
		return m.Start
	}
	return 0
}

func (m *StatisticsTask) GetEnd() int64 {
	if m != nil {
		return m.End
	}
	return 0
}

func (m *StatisticsTask) GetStatus() TaskStatus {
	if m != nil {
		return m.Status
	}
	return TaskStatus_TaskStatus_
}

func (m *StatisticsTask) GetMessage() string {
	if m != nil {
		return m.Message
	}
	return ""
}

func (m *StatisticsTask) GetCreateTime() int64 {
	if m != nil {
		return m.CreateTime
	}
	return 0
}

func (m *StatisticsTask) GetUpdateTime() int64 {
	if m != nil {
		return m.UpdateTime
	}
	return 0
}

type QueryById struct {
	Id                   int64    `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *QueryById) Reset()         { *m = QueryById{} }
func (m *QueryById) String() string { return proto.CompactTextString(m) }
func (*QueryById) ProtoMessage()    {}
func (*QueryById) Descriptor() ([]byte, []int) {
	return fileDescriptor_68a5e6cb4f7c8dc9, []int{11}
}
func (m *QueryById) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_QueryById.Unmarshal(m, b)
}
func (m *QueryById) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_QueryById.Marshal(b, m, deterministic)
}
func (m *QueryById) XXX_Merge(src proto.Message) {
	xxx_messageInfo_QueryById.Merge(m, src)
}
func (m *QueryById) XXX_Size() int {
	return xxx_messageInfo_QueryById.Size(m)
}
func (m *QueryById) XXX_DiscardUnknown() {
	xxx_messageInfo_QueryById.DiscardUnknown(m)
}

var xxx_messageInfo_QueryById proto.InternalMessageInfo

func (m *QueryById) GetId() int64 {
	if m != nil {
		return m.Id
	}
	return 0
}

func init() {
	proto.RegisterEnum("commons.sqlbuilder.ValueType", ValueType_name, ValueType_value)
	proto.RegisterEnum("commons.sqlbuilder.RangeType", RangeType_name, RangeType_value)
	proto.RegisterEnum("commons.sqlbuilder.TaskStatus", TaskStatus_name, TaskStatus_value)
	proto.RegisterType((*PageQuery)(nil), "commons.sqlbuilder.PageQuery")
	proto.RegisterType((*Value)(nil), "commons.sqlbuilder.Value")
	proto.RegisterType((*JoinCondition)(nil), "commons.sqlbuilder.JoinCondition")
	proto.RegisterType((*Condition)(nil), "commons.sqlbuilder.Condition")
	proto.RegisterType((*OrderCondition)(nil), "commons.sqlbuilder.OrderCondition")
	proto.RegisterType((*Query)(nil), "commons.sqlbuilder.Query")
	proto.RegisterType((*Update)(nil), "commons.sqlbuilder.Update")
	proto.RegisterType((*Field)(nil), "commons.sqlbuilder.Field")
	proto.RegisterType((*UpdateResult)(nil), "commons.sqlbuilder.UpdateResult")
	proto.RegisterType((*StatisticsTimeRange)(nil), "commons.sqlbuilder.StatisticsTimeRange")
	proto.RegisterType((*StatisticsTask)(nil), "commons.sqlbuilder.StatisticsTask")
	proto.RegisterType((*QueryById)(nil), "commons.sqlbuilder.QueryById")
}

func init() { proto.RegisterFile("builder.proto", fileDescriptor_68a5e6cb4f7c8dc9) }

var fileDescriptor_68a5e6cb4f7c8dc9 = []byte{
	// 1009 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x56, 0xdf, 0x72, 0xdb, 0xc4,
	0x17, 0xae, 0x24, 0xff, 0x91, 0x8e, 0x6c, 0x77, 0x7f, 0xdb, 0xdf, 0x85, 0x4b, 0xa7, 0xe0, 0x8a,
	0x5e, 0x78, 0x32, 0x43, 0x98, 0x86, 0x81, 0x8b, 0x0e, 0x5c, 0x24, 0x0d, 0x85, 0x30, 0x4d, 0x09,
	0x9b, 0x00, 0x03, 0x17, 0x78, 0x36, 0xd6, 0xd6, 0x88, 0xc8, 0xbb, 0x66, 0x77, 0x95, 0xc1, 0x7d,
	0x04, 0x86, 0xc7, 0xe0, 0x01, 0x78, 0x15, 0x5e, 0x80, 0x67, 0xe0, 0x11, 0x98, 0xb3, 0xab, 0xc8,
	0x4a, 0x6a, 0x0a, 0xdc, 0x9d, 0x3f, 0x9f, 0xce, 0x77, 0xf6, 0x9c, 0x6f, 0xd7, 0x86, 0xe1, 0x79,
	0x55, 0x94, 0xb9, 0xd0, 0xbb, 0x2b, 0xad, 0xac, 0xa2, 0x74, 0xae, 0x96, 0x4b, 0x25, 0xcd, 0xae,
	0xf9, 0xb1, 0xac, 0x33, 0xd9, 0x27, 0x90, 0x9c, 0xf0, 0x85, 0xf8, 0xa2, 0x12, 0x7a, 0x4d, 0xef,
	0x41, 0xb2, 0xe2, 0x0b, 0x31, 0x33, 0xc5, 0x4b, 0x31, 0x0e, 0x26, 0xc1, 0x34, 0x62, 0x31, 0x06,
	0x4e, 0x8b, 0x97, 0x82, 0xde, 0x07, 0x70, 0xc9, 0x42, 0xe6, 0xe2, 0xa7, 0x71, 0xe8, 0xb2, 0x0e,
	0x7e, 0x84, 0x81, 0xec, 0xb7, 0x10, 0xba, 0x5f, 0xf1, 0xb2, 0x12, 0xf4, 0x43, 0x80, 0x4b, 0x34,
	0x66, 0x76, 0xbd, 0xf2, 0x65, 0x46, 0x7b, 0xf7, 0x77, 0x5f, 0xe5, 0xde, 0x75, 0xf0, 0xb3, 0xf5,
	0x4a, 0xb0, 0xe4, 0xf2, 0xca, 0xc4, 0x1e, 0x0a, 0x69, 0x67, 0x2e, 0x50, 0xb3, 0xc4, 0x85, 0xb4,
	0xbe, 0xf4, 0x03, 0x18, 0xe4, 0xaa, 0x3a, 0x2f, 0x45, 0x9d, 0x8f, 0x26, 0xc1, 0x34, 0x60, 0xa9,
	0x8f, 0x35, 0x10, 0x63, 0x75, 0x21, 0x17, 0x35, 0xa4, 0x33, 0x09, 0xa6, 0x09, 0x4b, 0x7d, 0xcc,
	0x43, 0xee, 0x03, 0x9c, 0x2b, 0x55, 0xd6, 0x80, 0xee, 0x24, 0x98, 0xc6, 0x2c, 0xc1, 0x48, 0x93,
	0x6e, 0x3a, 0x30, 0xe3, 0xde, 0x24, 0xc2, 0x83, 0x5e, 0xb5, 0x60, 0xe8, 0xdb, 0x30, 0x6c, 0x13,
	0x98, 0x71, 0x3c, 0x89, 0xa6, 0x09, 0x1b, 0xb4, 0x18, 0x1c, 0xa8, 0xdd, 0xa8, 0x19, 0x27, 0x93,
	0x68, 0x1a, 0xb0, 0x41, 0xab, 0x53, 0x93, 0xfd, 0x1a, 0xc0, 0xf0, 0x33, 0x55, 0xc8, 0x27, 0x4a,
	0xe6, 0x85, 0x2d, 0x94, 0x44, 0x6a, 0xcb, 0xf1, 0x2b, 0xc9, 0x97, 0x7e, 0x74, 0x09, 0x4b, 0x5c,
	0xe4, 0x39, 0x5f, 0x0a, 0xfa, 0x10, 0x46, 0x3e, 0x3d, 0x57, 0xa5, 0x87, 0x84, 0x0e, 0x32, 0x70,
	0xd1, 0x27, 0xaa, 0x74, 0xa8, 0x77, 0xe0, 0xce, 0x0f, 0xaa, 0x90, 0xb3, 0x1b, 0xd0, 0xc8, 0x41,
	0x09, 0xa6, 0xce, 0xda, 0xf0, 0xb7, 0x20, 0xf5, 0x48, 0x5e, 0x16, 0x86, 0xd7, 0xf3, 0xf2, 0x6d,
	0xec, 0x63, 0x24, 0x33, 0x90, 0x6c, 0x3a, 0xbc, 0x0b, 0x71, 0x53, 0xd1, 0xf7, 0xd7, 0x9f, 0xd7,
	0x85, 0xde, 0x80, 0x58, 0xad, 0x84, 0xe6, 0x56, 0xe9, 0xba, 0xaf, 0xc6, 0xa7, 0xef, 0x42, 0x77,
	0xb3, 0xb1, 0x74, 0xef, 0xee, 0xdf, 0xca, 0x81, 0x79, 0x5c, 0xf6, 0x18, 0x46, 0x9f, 0xeb, 0x5c,
	0xe8, 0x0d, 0x33, 0x85, 0x4e, 0x8b, 0xd5, 0xd9, 0xf4, 0xff, 0xd0, 0x55, 0x88, 0xaa, 0xf9, 0xbc,
	0x93, 0xfd, 0x1e, 0x42, 0xd7, 0x0b, 0xfa, 0x23, 0x80, 0xf9, 0x55, 0x01, 0x33, 0x0e, 0x26, 0xd1,
	0x34, 0xdd, 0x2e, 0xc5, 0x86, 0x86, 0xb5, 0x3e, 0xa0, 0xc7, 0x40, 0x5c, 0xc5, 0x59, 0xab, 0x48,
	0xe8, 0x8a, 0x64, 0xdb, 0x8a, 0x5c, 0x6f, 0x98, 0xdd, 0x56, 0xd7, 0x7c, 0x43, 0x1f, 0x41, 0x07,
	0xef, 0x4b, 0x3d, 0x83, 0xad, 0x7d, 0x34, 0x77, 0x91, 0x39, 0x28, 0xaa, 0x59, 0x0b, 0x5b, 0x69,
	0x39, 0xb3, 0xca, 0xf2, 0xd2, 0x6d, 0x27, 0x66, 0xa9, 0x8f, 0x9d, 0x61, 0x08, 0x35, 0xf3, 0x42,
	0xe9, 0xb9, 0x98, 0xb9, 0xda, 0xb5, 0x9a, 0x5d, 0x04, 0x8b, 0xd1, 0xf7, 0xa1, 0x83, 0x2b, 0x1f,
	0xf7, 0x1c, 0xe9, 0x83, 0x6d, 0xa4, 0xd7, 0x34, 0xc8, 0x1c, 0x9c, 0x8e, 0x20, 0x5c, 0x5d, 0x8c,
	0xfb, 0x6e, 0xac, 0xe1, 0xea, 0x22, 0xfb, 0x39, 0x80, 0xde, 0x97, 0xab, 0x9c, 0x5b, 0x81, 0xa9,
	0x22, 0xaf, 0x9f, 0x87, 0xb0, 0xc8, 0xe9, 0x23, 0xe8, 0xbd, 0x28, 0x44, 0x99, 0x5f, 0xcd, 0x66,
	0xeb, 0x72, 0x9f, 0x22, 0x82, 0xd5, 0xc0, 0x1b, 0x7b, 0x89, 0xfe, 0xe3, 0x5e, 0xb2, 0xef, 0xa0,
	0xeb, 0xea, 0xbd, 0x4e, 0x8d, 0x8f, 0x21, 0x75, 0x64, 0xad, 0x97, 0xe4, 0xb5, 0xba, 0x03, 0x87,
	0x76, 0x76, 0xf6, 0x10, 0x06, 0xfe, 0xac, 0x4c, 0x98, 0xaa, 0xb4, 0x28, 0xb3, 0xb9, 0xaa, 0xa4,
	0xad, 0x0f, 0xed, 0x9d, 0xec, 0x97, 0x00, 0xee, 0x9c, 0x5a, 0x6e, 0x0b, 0x63, 0x8b, 0xb9, 0x39,
	0x2b, 0x96, 0x82, 0x71, 0xb9, 0x10, 0xb8, 0xe6, 0x7f, 0x7a, 0xf9, 0x1c, 0xd0, 0xbd, 0x7c, 0x0e,
	0x8a, 0x57, 0xa7, 0x90, 0x56, 0xe8, 0x4b, 0x5e, 0xb6, 0xde, 0x3c, 0xe7, 0x23, 0xb9, 0xb1, 0x5c,
	0x5b, 0x27, 0x9b, 0x88, 0x79, 0x87, 0x12, 0x88, 0x84, 0xcc, 0x9d, 0x1e, 0x22, 0x86, 0x66, 0xf6,
	0x67, 0x00, 0xa3, 0x56, 0x3b, 0xdc, 0x5c, 0xbc, 0xb2, 0x29, 0x0a, 0x1d, 0xcb, 0xcd, 0x45, 0x4d,
	0xe1, 0xec, 0x7f, 0x5b, 0x9e, 0x7e, 0x00, 0x3d, 0x63, 0xb9, 0xad, 0x8c, 0x93, 0xd8, 0x68, 0xef,
	0xcd, 0x6d, 0xe7, 0x42, 0xd6, 0x53, 0x87, 0x62, 0x35, 0x9a, 0x8e, 0xa1, 0xbf, 0x14, 0xc6, 0xa0,
	0x36, 0x7b, 0x7e, 0x43, 0xb5, 0x8b, 0x0f, 0xcf, 0x5c, 0x0b, 0x6e, 0xc5, 0xcc, 0x16, 0x4b, 0xe1,
	0xb4, 0x16, 0x31, 0xf0, 0x21, 0x9c, 0x26, 0x02, 0x2a, 0xb7, 0x06, 0x0f, 0x88, 0x3d, 0xc0, 0x87,
	0x10, 0x90, 0xdd, 0x83, 0xc4, 0x5d, 0x96, 0x83, 0xf5, 0x51, 0x7e, 0xf3, 0xb0, 0x3b, 0x7f, 0x04,
	0x90, 0x34, 0xbf, 0x30, 0x74, 0x04, 0xd0, 0x38, 0x33, 0x72, 0x8b, 0xf6, 0x21, 0x3a, 0x92, 0x96,
	0x04, 0x14, 0xa0, 0x77, 0xa8, 0xaa, 0x73, 0x51, 0x92, 0x10, 0xed, 0x53, 0xf7, 0x8a, 0x93, 0x88,
	0xc6, 0xd0, 0x39, 0x50, 0xaa, 0x24, 0x1d, 0xb4, 0x90, 0x8d, 0x74, 0xe9, 0x6d, 0x48, 0x7d, 0x7e,
	0x5f, 0x6b, 0xbe, 0x26, 0x3d, 0x3a, 0x80, 0xf8, 0x48, 0x5a, 0xef, 0xf5, 0x31, 0x7d, 0xe8, 0xde,
	0x77, 0x1f, 0x88, 0xe9, 0x10, 0x12, 0xfc, 0xd2, 0xbb, 0x89, 0x2f, 0x8f, 0x43, 0x21, 0x80, 0x5f,
	0x7a, 0x5b, 0x18, 0x92, 0x62, 0x77, 0x27, 0x8b, 0xa6, 0xd2, 0x80, 0xfe, 0x0f, 0x86, 0x27, 0x8b,
	0x36, 0xd5, 0x90, 0x12, 0x18, 0x9c, 0x2c, 0x9e, 0x96, 0x8a, 0xd7, 0xa0, 0xd1, 0xce, 0x0c, 0x92,
	0x46, 0x47, 0x78, 0x9e, 0xfd, 0x67, 0xcf, 0xc8, 0x2d, 0x24, 0x39, 0x2e, 0x64, 0x65, 0x05, 0x09,
	0x68, 0x0a, 0x7d, 0x6f, 0x3f, 0x27, 0x21, 0x1e, 0xe3, 0x53, 0x55, 0x69, 0x12, 0x21, 0xf6, 0x90,
	0xaf, 0xfd, 0xc9, 0xbe, 0x16, 0xe2, 0x82, 0x74, 0x69, 0x02, 0xdd, 0x63, 0x25, 0xed, 0xf7, 0xa4,
	0x87, 0xc1, 0x6f, 0x04, 0xd7, 0xa4, 0xbf, 0x73, 0x00, 0xb0, 0x59, 0x28, 0x9e, 0x6e, 0xe3, 0xe1,
	0x08, 0x53, 0xe8, 0xb3, 0x4a, 0x4a, 0x1c, 0x97, 0xa3, 0x3a, 0xad, 0xe6, 0x73, 0x61, 0x0c, 0x09,
	0xb1, 0xda, 0xc7, 0x5a, 0x2b, 0x4d, 0xa2, 0x83, 0xc1, 0xb7, 0xb0, 0xd1, 0xc7, 0x79, 0xcf, 0xfd,
	0x11, 0x79, 0xef, 0xaf, 0x00, 0x00, 0x00, 0xff, 0xff, 0x97, 0x97, 0xfb, 0x5d, 0x99, 0x08, 0x00,
	0x00,
}
