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

var HashServiceWithMd5 = NewHashService(md5.New)
var HashServiceWithSh1 = NewHashService(sha1.New)
var HashServiceWithSha512 = NewHashService(sha512.New)
var HashServiceWithSha256 = NewHashService(sha256.New)

// HashService  hash service
type HashService interface {
	Hash(data []byte) []byte
	HashToHexString(data []byte) string
}

type hashServiceImpl struct {
	pool sync.Pool // chan hash.Hash
}

// NewHashService new a HashService
func NewHashService(hashBuilder func() hash.Hash) HashService {
	pool := sync.Pool{
		New: func() interface{} {
			return hashBuilder()
		},
	}
	return &hashServiceImpl{
		pool: pool,
	}
}

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

func (impl *hashServiceImpl) HashToHexString(data []byte) string {
	return hex.EncodeToString(impl.Hash(data))
}
