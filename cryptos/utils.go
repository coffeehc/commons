package cryptos

import "math/big"

func HashToInt(data []byte) int64 {
	val := big.NewInt(0)
	for _, d := range data {
		val.Lsh(val, 8)
		val.Add(val, big.NewInt(int64(d)))
	}
	return val.Int64()
}
