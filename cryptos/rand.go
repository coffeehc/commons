package cryptos

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
)

// GetRandInt64 获取随机的64位整数
func GetRandInt64() int64 {
	bs := make([]byte, 8)
	_, err := rand.Read(bs)
	if err != nil {
		return GetRandInt64()
	}
	i := int64(binary.BigEndian.Uint64(bs))
	if i < 0 {
		return -1 * i
	}
	return i
}

// GetRandString 构建对象长度的随机字符串
func GetRandString(size int, encoding *base64.Encoding) string {
	bs := make([]byte, size)
	_, err := rand.Read(bs)
	if err != nil {
		return GetRandString(size, encoding)
	}
	return encoding.EncodeToString(bs)
}

// GetRandBytes 获取指定长度的 Bytes
func GetRandBytes(size int) []byte {
	bs := make([]byte, size)
	_, err := rand.Read(bs)
	if err != nil {
		return GetRandBytes(size)
	}
	return bs
}

func GetRandInt(max, min int) int {
	v := int(GetRandInt64()) % (max - min)
	return v + min
}
