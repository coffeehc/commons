package coder

import (
	"google.golang.org/protobuf/types/known/wrapperspb"
	"testing"
)

func TestNumericDecodersRejectMalformedBytesAndNilTargets(t *testing.T) {
	for _, row := range []struct {
		codec  Coder
		target any
		size   int
	}{{Int64Coder, new(int64), 8}, {Uint64Coder, new(uint64), 8}, {Int32Coder, new(int32), 4}, {Uint32Coder, new(uint32), 4}} {
		for _, data := range [][]byte{nil, {1}, make([]byte, row.size+1)} {
			if err := row.codec.Unmarshal(data, row.target); err == nil {
				t.Fatal("accepted malformed numeric bytes")
			}
		}
	}
	var n *int64
	if err := Int64Coder.Unmarshal(make([]byte, 8), n); err == nil {
		t.Fatal("nil numeric target")
	}
	var s *string
	if err := StringCoder.Unmarshal(nil, s); err == nil {
		t.Fatal("nil string target")
	}
	var b *[]byte
	if err := ByteCoder.Unmarshal(nil, b); err == nil {
		t.Fatal("nil byte target")
	}
}
func TestProtobufCoderRejectsTypedNil(t *testing.T) {
	var message *wrapperspb.StringValue
	if _, err := PBCoder.Marshal(message); err == nil {
		t.Fatal("typed nil accepted")
	}
	if err := PBCoder.Unmarshal(nil, message); err == nil {
		t.Fatal("typed nil target accepted")
	}
}
