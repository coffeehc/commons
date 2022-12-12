package localdbservice

import (
	"bytes"
	"encoding/binary"
)

func BuildStorageKey(prefix []byte, extends ...[]byte) []byte {
	keys := make([][]byte, 0, len(extends)+len(Separator))
	keys = append(keys, prefix)
	keys = append(keys, extends...)
	key := bytes.Join(keys, Separator)
	return key
}

func KeyUpperBound(b []byte) []byte {
	end := make([]byte, len(b))
	copy(end, b)
	for i := len(end) - 1; i >= 0; i-- {
		end[i] = end[i] + 1
		if end[i] != 0 {
			return end[:i+1]
		}
	}
	return nil // no upper-bound
}
func KeyLowerBound(b []byte) []byte {
	end := make([]byte, len(b))
	copy(end, b)
	for i := len(end) - 1; i >= 0; i-- {
		if end[i] != 0 {
			continue
		}
		end[i] = end[i] - 1
		return end[:i+1]

	}
	return nil // no upper-bound
}

func BuildIdKey(prefix []byte, id int64) []byte {
	key := append(prefix, Separator...)
	key = append(key, DBInt64ToBytes(id)...)
	return key
}

func ParasIdKey(prefix []byte, key []byte) int64 {
	index := len(prefix) + len(Separator)
	data := key[index:]
	return DBBytesToInt64(data)
}

func DBInt64ToBytes(i int64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return buf
}

func DBBytesToInt64(buf []byte) int64 {
	i := binary.BigEndian.Uint64(buf)
	return int64(i)
}
