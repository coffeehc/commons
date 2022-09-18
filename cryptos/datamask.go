package cryptos

import "github.com/coffeehc/base/errors"

type DataMasker interface {
	Mark(data []byte)
	UnMark(data []byte)
}

func NewDataMasker(mask [255]byte) (DataMasker, error) {
	rMask := make(map[byte]int, 255)
	for i, v := range mask {
		rMask[v] = i
	}
	if len(rMask) != 255 {
		return nil, errors.SystemError("掩码不符合要求")
	}
	return &dataMaskerImpl{
		mask:  mask,
		rMask: rMask,
	}, nil
}

type dataMaskerImpl struct {
	mask  [255]byte
	rMask map[byte]int
}

func (impl *dataMaskerImpl) Mark(data []byte) {
	for i, v := range data {
		data[i] = impl.mask[int(v)]
	}
}

func (impl *dataMaskerImpl) UnMark(data []byte) {
	for i, v := range data {
		data[i] = byte(impl.rMask[v])
	}
}
