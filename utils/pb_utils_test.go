package utils

import (
	"testing"

	"github.com/coffeehc/commons/models"
	"google.golang.org/protobuf/proto"
)

func TestParsePayloadResponseDecodesPayload(t *testing.T) {
	want := &models.Error{Message: "payload", ErrorCode: 7}
	payload, err := proto.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	got := &models.Error{}
	if responseErr := ParsePayloadResponse(&models.PayloadResponse{Payload: payload}, got); responseErr != nil {
		t.Fatalf("valid payload should not return an error: %v", responseErr)
	}
	if !proto.Equal(got, want) {
		t.Fatalf("unexpected decoded payload: got=%v want=%v", got, want)
	}
}

func TestParsePayloadResponseReturnsResponseError(t *testing.T) {
	want := &models.Error{Message: "business failure", ErrorCode: 9}
	got := ParsePayloadResponse(&models.PayloadResponse{Err: want}, &models.Error{})
	if got != want {
		t.Fatalf("response error should pass through unchanged: got=%v want=%v", got, want)
	}
}

func TestParsePayloadResponseReturnsDecodeError(t *testing.T) {
	got := ParsePayloadResponse(&models.PayloadResponse{Payload: []byte{0xff}}, &models.Error{})
	if got == nil || got.Message == "" {
		t.Fatalf("invalid protobuf payload should return a model error: %v", got)
	}
}
