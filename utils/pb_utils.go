package utils

import (
	"errors"
	"github.com/coffeehc/commons/models"
	"reflect"
	"time"

	"google.golang.org/protobuf/proto"
)

func BuildError(message string) *models.Error {
	return &models.Error{
		Message: message,
	}
}

func ParsePayloadResponse(resp *models.PayloadResponse, payload proto.Message) *models.Error {
	err := resp.GetErr()
	if err != nil {
		return err
	}
	err1 := proto.Unmarshal(resp.GetPayload(), payload)
	panic(err1)
	return nil
}

func EncodeCacheSingleBytes(data []byte, exp time.Duration) ([]byte, error) {
	cacheSingle := &models.CacheSingle{
		Data: data,
		Exp:  time.Now().Add(exp).UnixNano(),
	}
	return proto.Marshal(cacheSingle)
}

func EncodeCacheSingleProtoBuf(message proto.Message, exp time.Duration) ([]byte, error) {
	data, err := proto.Marshal(message)
	if err != nil {
		return nil, err
	}
	return EncodeCacheSingleBytes(data, exp)
}

func DecodeCacheSingleData(data []byte) ([]byte, error) {
	cacheSingle := &models.CacheSingle{}
	err := proto.Unmarshal(data, cacheSingle)
	if err != nil {
		return nil, err
	}
	if time.Since(time.Unix(0, cacheSingle.GetExp())) > 0 {
		return nil, errors.New("数据已过期")
	}
	return cacheSingle.GetData(), nil
}

func DecodeCacheSingleProtoBuf(data []byte, message proto.Message) error {
	data, err := DecodeCacheSingleData(data)
	if err != nil {
		return err
	}
	return proto.Unmarshal(data, message)
}

// 该方法不做强制校验，异常直接再开发时就抛出
func EncodeCacheRepeatedProtoBuf(messages interface{}, exp time.Duration) ([]byte, error) {
	messagesValue := reflect.Indirect(reflect.ValueOf(messages))
	if messagesValue.Kind() != reflect.Slice {
		return nil, errors.New("需要序列化的数据不是数组")
	}
	messageCount := messagesValue.Len()
	datas := make([][]byte, messageCount)
	for i := 0; i < messageCount; i++ {
		data, err := proto.Marshal(messagesValue.Index(i).Interface().(proto.Message))
		if err != nil {
			return nil, err
		}
		datas[i] = data
	}
	cacheRepeated := &models.CacheRepeated{
		Datas: datas,
		Exp:   time.Now().Add(exp).UnixNano(),
	}
	return proto.Marshal(cacheRepeated)
}

func DecodeCacheRepeatedProtoBuf(data []byte, messagesPtr interface{}) error {
	cacheRepeated := &models.CacheRepeated{}
	err := proto.Unmarshal(data, cacheRepeated)
	if err != nil {
		return err
	}
	if time.Since(time.Unix(0, cacheRepeated.GetExp())) > 0 {
		return errors.New("数据已过期")
	}
	datas := cacheRepeated.GetDatas()
	sliceValue := reflect.Indirect(reflect.ValueOf(messagesPtr))
	sliceElementType := sliceValue.Type().Elem().Elem()
	for _, data := range datas {
		message := reflect.New(sliceElementType)
		err = proto.Unmarshal(data, message.Interface().(proto.Message))
		if err != nil {
			return err
		}
		sliceValue.Set(reflect.Append(sliceValue, message))

	}
	return nil
}

func ProtoMarshal(message proto.Message) []byte {
	data, _ := proto.Marshal(message)
	return data
}

func ProtoUnMarshal(data []byte, message proto.Message) error {
	return proto.Unmarshal(data, message)
}
