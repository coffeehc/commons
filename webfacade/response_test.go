package webfacade

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
)

func TestAjaxEnvelopeWireFixtures(t *testing.T) {
	for _, test := range []struct {
		value AjaxResponse
		want  string
	}{
		{AjaxResponse{}, `{"success":false,"message":"","payload":null,"request_id":"","code":0}`},
		{AjaxResponse{Success: true, Code: 0, Payload: map[string]string{"id": "one"}}, `{"success":true,"message":"","payload":{"id":"one"},"request_id":"","code":0}`},
		{AjaxResponse{Success: true, Code: 1, Message: "ok", RequestID: "request-1", Payload: []string{}}, `{"success":true,"message":"ok","payload":[],"request_id":"request-1","code":1}`},
		{AjaxResponse{Code: 40100, Message: "denied", Redirect: "/login"}, `{"success":false,"message":"denied","payload":null,"request_id":"","code":40100,"redirect":"/login"}`},
	} {
		current := test.value
		var response BaseResponse = &current
		if response.IsSuccess() != current.Success || response.GetMessage() != current.Message || response.GetCode() != current.Code || response.GetRequestId() != current.RequestID {
			t.Fatal("accessor contract changed")
		}
		got, err := json.Marshal(current)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != test.want {
			t.Fatalf("got %s; want %s", got, test.want)
		}
	}
}

func TestSharedListFixtures(t *testing.T) {
	type fixture struct{ Name, Response, Normalized string }
	var fixtures struct {
		Contract               string
		Valid, Legacy, Invalid []fixture
	}
	data, err := os.ReadFile("testdata/webfacade-list.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	if fixtures.Contract != "webfacade-list.v1" {
		t.Fatal(fixtures.Contract)
	}
	decode := func(raw string) (AjaxResponse, ListData[map[string]any], error) {
		var envelope AjaxResponse
		var payload struct {
			Payload json.RawMessage `json:"payload"`
		}
		var list ListData[map[string]any]
		if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
			return envelope, list, err
		}
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			return envelope, list, err
		}
		err := json.Unmarshal(payload.Payload, &list)
		return envelope, list, err
	}
	for _, item := range append(fixtures.Valid, fixtures.Legacy...) {
		t.Run(item.Name, func(t *testing.T) {
			envelope, list, err := decode(item.Response)
			if err != nil {
				t.Fatal(err)
			}
			if list.List == nil {
				t.Fatal("list must normalize to array")
			}
			var raw struct {
				Payload map[string]json.RawMessage `json:"payload"`
			}
			if err := json.Unmarshal([]byte(item.Response), &raw); err != nil {
				t.Fatal(err)
			}
			_, cursorPresent := raw.Payload["next_cursor"]
			if (list.NextCursor != nil) != cursorPresent {
				t.Fatal("cursor presence lost")
			}
			envelope.Payload = list
			got, err := json.Marshal(envelope)
			if err != nil {
				t.Fatal(err)
			}
			want := item.Response
			if item.Normalized != "" {
				want = item.Normalized
			}
			if string(got) != want {
				t.Fatalf("got %s; want %s", got, want)
			}
		})
	}
	for _, item := range fixtures.Invalid {
		t.Run(item.Name, func(t *testing.T) {
			if _, _, err := decode(item.Response); err == nil {
				t.Fatal("invalid payload accepted")
			}
		})
	}
}

func TestListProducerValidationAndCursorStates(t *testing.T) {
	empty, err := json.Marshal(ListData[string]{})
	if err != nil || string(empty) != `{"list":[]}` {
		t.Fatalf("%s: %v", empty, err)
	}
	for _, value := range []int64{-1, MaxSafeInteger + 1} {
		if _, err := json.Marshal(ListData[string]{Total: &value}); err == nil {
			t.Fatal("unsafe total accepted")
		}
	}
	for _, token := range []string{"", strings.Repeat("x", MaxCursorBytes+1), string([]byte{0xff})} {
		if _, err := NextPageCursor(token); err == nil {
			t.Fatal("invalid cursor accepted")
		}
	}
	next, err := NextPageCursor(" opaque ")
	if err != nil {
		t.Fatal(err)
	}
	if token, ok := next.Token(); !ok || token != " opaque " {
		t.Fatal("opaque token changed")
	}
	if _, ok := EndCursor().Token(); ok {
		t.Fatal("end cursor has a token")
	}
	var target ListData[string]
	if err := json.Unmarshal([]byte(`{"list":[],"next_cursor":null}`), &target); err != nil || target.NextCursor == nil {
		t.Fatalf("terminal cursor lost: %+v %v", target, err)
	}
	if err := json.Unmarshal([]byte(`{"list":[]}`), &target); err != nil || target.NextCursor != nil {
		t.Fatal("omitted cursor retained stale state")
	}
}

func TestPBWireRemainsCompatible(t *testing.T) {
	current := &PBResponse{Success: true, Code: 17, Message: "ok", RequestId: 31, Payload: []byte("payload")}
	// Fixed wire fixture keeps this contract independent of the deleted legacy API.
	want := []byte{0x08, 17, 0x12, 2, 'o', 'k', 0x18, 31, 0x20, 1, 0x2a, 7, 'p', 'a', 'y', 'l', 'o', 'a', 'd'}
	got, err := proto.Marshal(current)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("protobuf wire changed: %x != %x", got, want)
	}
}

func TestEmptyFilteredListMayContinueCursorScan(t *testing.T) {
	var page ListData[string]
	if err := json.Unmarshal([]byte(`{"list":[],"next_cursor":"scan-after-filtered-row"}`), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.List) != 0 || page.NextCursor == nil {
		t.Fatal("empty page lost continuation")
	}
	if token, more := page.NextCursor.Token(); !more || token != "scan-after-filtered-row" {
		t.Fatal("cursor completion inferred from list length")
	}
}
