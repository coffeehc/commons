package cryptos

import (
	"crypto/sha512"
	"hash"
)

var (
	//SHA512HashService sha512的 hash 服务,默认支持100并发hash 请求
	SHA512HashService = NewHashService(func() hash.Hash { return sha512.New() }, 100)
)

//HashService  hash service
type HashService interface {
	Hash(data []byte) []byte
}

type _HashService struct {
	pool chan hash.Hash
}

//NewHashService new a HashService
func NewHashService(hashBuilder func() hash.Hash, concurrent int) HashService {
	pool := make(chan hash.Hash, concurrent)
	for i := 0; i < concurrent; i++ {
		pool <- hashBuilder()
	}
	return &_HashService{
		pool: pool,
	}
}

func (service *_HashService) Hash(data []byte) []byte {
	hashImpl := <-service.pool
	defer func() {
		hashImpl.Reset()
		service.pool <- hashImpl
	}()
	hashImpl.Write(data)
	return hashImpl.Sum(nil)
}
