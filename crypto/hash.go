package crypto

import (
	"hash"
	"crypto/md5"
)

var (
	HashBudiler_md5 = func()hash.Hash {return md5.New()}
)


type HashService struct {
	pool chan hash.Hash
}

func NewHashService(hashBuilder func()hash.Hash,concurrent int){
	pool := make(chan hash.Hash,concurrent)
	for i:=0;i<concurrent;i++{
		pool <- hashBuilder()
	}
	return &HashService{
		pool:pool,
	}
}

func (this *HashService)Hash(data []byte)[]byte{
	hashImpl :=  <- this.pool
	defer func(){
		hashImpl.Reset()
		this.pool <- hashImpl
	}()
	hashImpl.Write(data)
	return hashImpl.Sum(nil)
}
