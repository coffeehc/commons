package cryptos

import (
	"encoding/hex"
	"hash"
)

//HashService  hash service
type HashService interface {
	Hash(data []byte) []byte
	HashToHexString(data []byte) string
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

func (service *_HashService) HashToHexString(data []byte) string {
	return hex.EncodeToString(service.Hash(data))
}
