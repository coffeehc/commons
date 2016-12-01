package binarys

import "encoding/base64"

var Base64Codec = _Base64Codec{}

type BytesCodec interface {
	Encode([]byte)string
	Decode(string)([]byte,error)
}

type _Base64Codec struct {
}

func (this _Base64Codec)Encode(data []byte) string{
	return base64.RawStdEncoding.EncodeToString(data)
}

func (this _Base64Codec)Decode(str string)([]byte,error)  {
	return base64.RawStdEncoding.DecodeString(str)
}
