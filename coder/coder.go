package coder

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"github.com/gogo/protobuf/proto"
)

type Coder interface {
	Marshal(obj interface{}) ([]byte, error)
	Unmarshal([]byte, interface{}) error
}

var (
	ByteCoder   = byteCoder{}
	StringCoder = stringCoder{}
	JsonCoder   = jsonCoder{}
	PBCoder     = pbCoder{}
	Uint64Coder = uint64Coder{}
	Int64Coder  = int64Coder{}
	Uint32Coder = uint32Coder{}
	Int32Coder  = int32Coder{}
)

type byteCoder struct {
}

func (byteCoder) Marshal(obj interface{}) ([]byte, error) {
	data, ok := obj.([]byte)
	if ok {
		return data, nil
	}
	return nil, errors.New("对象不是[]byte类型")
}
func (byteCoder) Unmarshal(data []byte, target interface{}) error {
	t, ok := target.(*[]byte)
	if ok {
		*t = make([]byte, len(data))
		copy(*t, data)
		return nil
	}
	return errors.New("target不是[]byte类型")
}

type stringCoder struct {
}

func (stringCoder) Marshal(obj interface{}) ([]byte, error) {
	data, ok := obj.(string)
	if ok {
		return []byte(data), nil
	}
	return nil, errors.New("对象不是string类型")
}
func (stringCoder) Unmarshal(data []byte, target interface{}) error {
	t, ok := target.(*string)
	if ok {
		*t = string(data)
		return nil
	}
	return errors.New("target不是string类型")
}

type jsonCoder struct{}

func (jsonCoder) Marshal(obj interface{}) ([]byte, error) {
	return json.Marshal(obj)
}
func (jsonCoder) Unmarshal(data []byte, target interface{}) error {
	return json.Unmarshal(data, target)
}

type pbCoder struct{}

func (pbCoder) Marshal(obj interface{}) ([]byte, error) {
	msg, ok := obj.(proto.Message)
	if ok {
		return proto.Marshal(msg)
	}
	return nil, errors.New("对象不是protobuf.Message类型")

}
func (pbCoder) Unmarshal(data []byte, target interface{}) error {
	msg, ok := target.(proto.Message)
	if ok {
		return proto.Unmarshal(data, msg)
	}
	return errors.New("对象不是protobuf.Message类型")
}

type uint64Coder struct{}

func (uint64Coder) Marshal(obj interface{}) ([]byte, error) {
	i, ok := obj.(uint64)
	if ok {
		data := make([]byte, 8)
		binary.BigEndian.PutUint64(data, i)
		return data, nil
	}
	return nil, errors.New("对象不是uint64类型")

}
func (uint64Coder) Unmarshal(data []byte, target interface{}) error {
	iptr, ok := target.(*uint64)
	if ok {
		*iptr = binary.BigEndian.Uint64(data)
		return nil
	}
	return errors.New("对象不是*uint64类型")
}

type int64Coder struct{}

func (int64Coder) Marshal(obj interface{}) ([]byte, error) {
	i, ok := obj.(int64)
	if ok {
		data := make([]byte, 8)
		binary.BigEndian.PutUint64(data, uint64(i))
		return data, nil
	}
	return nil, errors.New("对象不是int64类型")

}
func (int64Coder) Unmarshal(data []byte, target interface{}) error {
	iptr, ok := target.(*int64)
	if ok {
		*iptr = int64(binary.BigEndian.Uint64(data))
		return nil
	}
	return errors.New("对象不是*int64类型")
}

type uint32Coder struct{}

func (uint32Coder) Marshal(obj interface{}) ([]byte, error) {
	i, ok := obj.(uint32)
	if ok {
		data := make([]byte, 4)
		binary.BigEndian.PutUint32(data, i)
		return data, nil
	}
	return nil, errors.New("对象不是uint64类型")

}
func (uint32Coder) Unmarshal(data []byte, target interface{}) error {
	iptr, ok := target.(*uint32)
	if ok {
		*iptr = binary.BigEndian.Uint32(data)
		return nil
	}
	return errors.New("对象不是*uint64类型")
}

type int32Coder struct{}

func (int32Coder) Marshal(obj interface{}) ([]byte, error) {
	i, ok := obj.(int32)
	if ok {
		data := make([]byte, 4)
		binary.BigEndian.PutUint32(data, uint32(i))
		return data, nil
	}
	return nil, errors.New("对象不是int64类型")

}
func (int32Coder) Unmarshal(data []byte, target interface{}) error {
	iptr, ok := target.(*int32)
	if ok {
		*iptr = int32(binary.BigEndian.Uint32(data))
		return nil
	}
	return errors.New("对象不是*int64类型")
}
