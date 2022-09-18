package cryptos

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
)

// AesCBCEncrypt
func AesCBCEncrypt(src, key, iv []byte, padding string) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	return CBCEncrypt(block, src, iv, padding)
}

// AesCBCDecrypt
func AesCBCDecrypt(src, key, iv []byte, padding string) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	return CBCDecrypt(block, src, iv, padding)
}

// CBCEncrypt
func CBCEncrypt(block cipher.Block, src, iv []byte, padding string) ([]byte, error) {
	blockSize := block.BlockSize()
	src = Padding(padding, src, blockSize)

	encryptData := make([]byte, len(src))

	if len(iv) != block.BlockSize() {
		return nil, errors.New("CBCEncrypt: IV length must equal block size")
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(encryptData, src)

	return encryptData, nil
}

// CBCDecrypt
func CBCDecrypt(block cipher.Block, src, iv []byte, padding string) ([]byte, error) {

	dst := make([]byte, len(src))

	if len(iv) != block.BlockSize() {
		return nil, errors.New("CBCDecrypt: IV length must equal block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(dst, src)

	return UnPadding(padding, dst)
}

var ErrUnPadding = errors.New("UnPadding error")

const PKCS5_PADDING = "PKCS5"
const PKCS7_PADDING = "PKCS7"
const ZEROS_PADDING = "ZEROS"

func Padding(padding string, src []byte, blockSize int) []byte {
	switch padding {
	case PKCS5_PADDING:
		src = PKCS5Padding(src, blockSize)
	case PKCS7_PADDING:
		src = PKCS7Padding(src, blockSize)
	case ZEROS_PADDING:
		src = ZerosPadding(src, blockSize)
	}
	return src
}

func UnPadding(padding string, src []byte) ([]byte, error) {
	switch padding {
	case PKCS5_PADDING:
		return PKCS5Unpadding(src)
	case PKCS7_PADDING:
		return PKCS7UnPadding(src)
	case ZEROS_PADDING:
		return ZerosUnPadding(src)
	}
	return src, nil
}

func PKCS5Padding(src []byte, blockSize int) []byte {
	return PKCS7Padding(src, blockSize)
}

func PKCS5Unpadding(src []byte) ([]byte, error) {
	return PKCS7UnPadding(src)
}

func PKCS7Padding(src []byte, blockSize int) []byte {
	padding := blockSize - len(src)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padtext...)
}

func PKCS7UnPadding(src []byte) ([]byte, error) {
	length := len(src)
	if length == 0 {
		return src, ErrUnPadding
	}
	unpadding := int(src[length-1])
	if length < unpadding {
		return src, ErrUnPadding
	}
	return src[:(length - unpadding)], nil
}

func ZerosPadding(src []byte, blockSize int) []byte {
	paddingCount := blockSize - len(src)%blockSize
	if paddingCount == 0 {
		return src
	} else {
		return append(src, bytes.Repeat([]byte{byte(0)}, paddingCount)...)
	}
}

func ZerosUnPadding(src []byte) ([]byte, error) {
	for i := len(src) - 1; ; i-- {
		if src[i] != 0 {
			return src[:i+1], nil
		}
	}
}
