package cryptos

import (
	"crypto/aes"
	"encoding/base64"

	"github.com/coffeehc/base/errors"
	"github.com/coffeehc/base/log"
	"go.uber.org/zap"
)

func DataEncode(data []byte, mask DataMasker) (string, error) {
	key := GetRandBytes(16)
	vi := GetRandBytes(aes.BlockSize)
	dist, err := AesCBCEncrypt(data, key, vi, PKCS7_PADDING)
	if err != nil {
		log.Warn("AESEncrypt失败", zap.Error(err))
		return "", errors.ConverError(err)
	}
	if mask != nil {
		mask.Mark(key)
	}
	dist = append(vi, dist...)
	distLen := len(dist) / 2
	sourceData := make([]byte, 4+16+distLen*2)
	tokenData := sourceData[4:]
	copy(tokenData, key[:4])
	copy(tokenData[4:], dist[:distLen])
	copy(tokenData[4+distLen:], key[4:12])
	copy(tokenData[12+distLen:], dist[distLen:])
	copy(tokenData[12+distLen*2:], key[12:])
	return base64.RawURLEncoding.EncodeToString(sourceData), nil
}

func DataDecode(token string, mask DataMasker) ([]byte, error) {
	sourceData, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		log.Warn("解析Token失败", zap.Error(err))
		return nil, errors.MessageError("token非法")
	}
	dist := sourceData[4:]
	key := make([]byte, 16)
	data := make([]byte, len(dist)-16)
	dataLen := len(data) / 2
	copy(key, dist[:4])
	copy(key[4:], dist[4+dataLen:12+dataLen])
	copy(key[12:], dist[12+dataLen*2:])
	copy(data, dist[4:4+dataLen])
	copy(data[dataLen:], dist[12+dataLen:])
	if mask != nil {
		mask.UnMark(key)
	}

	data, err = AesCBCDecrypt(data[aes.BlockSize:], key, data[:aes.BlockSize], PKCS7_PADDING)
	if err != nil {
		log.Warn("Token解码失败", zap.Error(err))
		return nil, errors.MessageError("token非法")
	}
	return data, nil
}
