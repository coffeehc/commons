package cryptos

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"hash"
	"sync"
)

var (
	// HashServiceWithMd5 提供 MD5 摘要计算，供兼容既有非安全哈希场景使用。
	HashServiceWithMd5 = NewHashService(md5.New)
	// HashServiceWithSh1 提供 SHA-1 摘要计算，名称保留既有公开接口拼写。
	HashServiceWithSh1 = NewHashService(sha1.New)
	// HashServiceWithSha512 提供 SHA-512 摘要计算。
	HashServiceWithSha512 = NewHashService(sha512.New)
	// HashServiceWithSha256 提供 SHA-256 摘要计算。
	HashServiceWithSha256 = NewHashService(sha256.New)
)

// HashService 定义可并发复用的字节摘要计算能力。
// 实现不会保留输入数据，返回的摘要由调用方持有。
type HashService interface {
	// Hash 返回 data 的原始摘要字节。
	Hash(data []byte) []byte
	// HashToHexString 返回 data 摘要的小写十六进制编码。
	HashToHexString(data []byte) string
}

// hashServiceImpl 通过 sync.Pool 复用同一种摘要算法实例。
type hashServiceImpl struct {
	// pool 保存已经 Reset 的 hash.Hash 实例。
	pool sync.Pool
}

// NewHashService 使用 hashBuilder 创建可并发使用的摘要服务。
func NewHashService(hashBuilder func() hash.Hash) HashService {
	impl := &hashServiceImpl{}
	impl.pool.New = func() any {
		return hashBuilder()
	}
	return impl
}

// Hash 返回 data 的原始摘要字节，并在计算后回收摘要实例。
func (impl *hashServiceImpl) Hash(data []byte) []byte {
	v := impl.pool.Get()
	hashImpl := v.(hash.Hash)
	defer func() {
		hashImpl.Reset()
		impl.pool.Put(hashImpl)
	}()
	hashImpl.Write(data)
	return hashImpl.Sum(nil)
}

// HashToHexString 返回 data 摘要的小写十六进制编码。
func (impl *hashServiceImpl) HashToHexString(data []byte) string {
	return hex.EncodeToString(impl.Hash(data))
}
