package utils

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func Int64IdsDecode(idStrs []string, ignoreZero bool) []int64 {
	ids := make([]int64, 0, len(idStrs))
	for _, idStr := range idStrs {
		id := Int64IdDecode(idStr)
		if id == 0 && ignoreZero {
			continue
		}
		ids = append(ids, id)
	}
	return ids
}

func Int64IdEncode(id int64) string {
	return strconv.FormatInt(id, 16)
}

func Int64IdDecode(idStr string) int64 {
	id, _ := strconv.ParseInt(idStr, 16, 64)
	return id
}

func IpToInt64(ip string) (int64, error) {
	ips := strings.Split(ip, ".")
	E := errors.New("Not A IP.")
	if len(ips) != 4 {
		return 0, E
	}
	var intIP int64
	for k, v := range ips {
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil || i > 255 {
			return 0, E
		}
		intIP = intIP | i<<uint(8*(3-k))
	}
	return intIP, nil
}

func Int64ToIp(intIp int64) (string, error) {
	i4 := intIp & 255
	i3 := intIp >> 8 & 255
	i2 := intIp >> 16 & 255
	i1 := intIp >> 24 & 255
	if i1 > 255 || i2 > 255 || i3 > 255 || i4 > 255 {
		return "", errors.New("Isn't a IntIP Type.")
	}
	ipstring := fmt.Sprintf("%d.%d.%d.%d", i1, i2, i3, i4)
	return ipstring, nil
}

func Int64ToBytes(i int64) []byte {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, uint64(i))
	return data
}

func BytesToInt64(data []byte) (int64, error) {
	if len(data) != 8 {
		return 0, errors.New("无法解析")
	}
	return int64(binary.LittleEndian.Uint64(data)), nil
}

func BytesToBase64(seg []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(seg), "=")
}

// Decode JWT specific base64url encoding with padding stripped
func Base64ToByte(seg string) ([]byte, error) {
	if l := len(seg) % 4; l > 0 {
		seg += strings.Repeat("=", 4-l)
	}

	return base64.URLEncoding.DecodeString(seg)
}

func BytesToBase64WithRaw(seg []byte) string {
	return strings.TrimRight(base64.RawStdEncoding.EncodeToString(seg), "=")
}

// Decode JWT specific base64url encoding with padding stripped
func Base64ToByteWithRaw(seg string) ([]byte, error) {
	if l := len(seg) % 4; l > 0 {
		seg += strings.Repeat("=", 4-l)
	}

	return base64.RawStdEncoding.DecodeString(seg)
}
