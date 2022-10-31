package memcache

import (
	"encoding/binary"
	"encoding/json"

	"github.com/coffeehc/base/errors"
	"github.com/coffeehc/base/log"
	"github.com/gogo/protobuf/proto"
)

var (
	JsonCoder     = &jsonCoder{}
	ProtobufCoder = &protobufCoder{}
	StringCoder   = &stringCoder{}
	Int64Coder    = &int64Coder{}
)

type Coder interface {
	Marshal(target interface{}) ([]byte, error)
	Unmarshal(data []byte, target interface{}) error
}

type int64Coder struct{}

func (impl int64Coder) Marshal(target interface{}) ([]byte, error) {
	msg, ok := target.(uint64)
	if !ok {
		return nil, errors.MessageError("不是Protobuf类型")
	}
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, uint64(msg))
	return data, nil
}
func (impl int64Coder) Unmarshal(data []byte, target interface{}) error {
	msg, ok := target.(*int64)
	if !ok {
		return errors.MessageError("对象必须是字符串指针")
	}
	v := binary.BigEndian.Uint64(data)
	*msg = int64(v)
	return nil
}

type stringCoder struct{}

func (impl stringCoder) Marshal(target interface{}) ([]byte, error) {
	msg, ok := target.(string)
	if !ok {
		return nil, errors.MessageError("不是Protobuf类型")
	}
	return []byte(msg), nil
}
func (impl stringCoder) Unmarshal(data []byte, target interface{}) error {
	msg, ok := target.(*string)
	if !ok {
		return errors.MessageError("对象必须是字符串指针")
	}
	*msg = string(data)
	return nil
}

type jsonCoder struct{}

func (impl jsonCoder) Marshal(target interface{}) ([]byte, error) {
	return json.Marshal(target)
}
func (impl jsonCoder) Unmarshal(data []byte, target interface{}) error {
	return json.Unmarshal(data, target)
}

type protobufCoder struct{}

func (impl protobufCoder) Marshal(target interface{}) ([]byte, error) {
	msg, ok := target.(proto.Message)
	if !ok {
		return nil, errors.MessageError("不是Protobuf类型")
	}
	return proto.Marshal(msg)
}
func (impl protobufCoder) Unmarshal(data []byte, target interface{}) error {
	msg, ok := target.(proto.Message)
	if !ok {
		log.Error("对象不是proto.Message类型")
		return errors.MessageError("对象不是proto.Message类型")
	}
	return proto.Unmarshal(data, msg)
}
