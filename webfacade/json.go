package webfacade

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin/binding"
)

var jsonContentType = "application/json; charset=utf-8"
var jsonpContentType = "application/javascript; charset=utf-8"

type JsonRender struct {
	Data interface{}
}

// Render (JSON) writes data with custom ContentType.
func (r JsonRender) Render(w http.ResponseWriter) (err error) {
	if err = writeJSON(w, r.Data); err != nil {
		panic(err)
	}
	return
}

// WriteContentType (JSON) writes JSON ContentType.
func (r JsonRender) WriteContentType(w http.ResponseWriter) {
	WriteContentType(w, []string{jsonContentType})
}

// WriteJSON marshals the given interface object and writes it with custom ContentType.
func writeJSON(w http.ResponseWriter, obj interface{}) error {
	WriteContentType(w, []string{jsonContentType})
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	_, err = w.Write(jsonBytes)
	return err
}

var JsonBinding = jsonBinding{}

// -----------

var EnableDecoderUseNumber = false

type jsonBinding struct{}

func (jsonBinding) Name() string {
	return "json"
}

func (jsonBinding) Bind(req *http.Request, obj interface{}) error {
	if req == nil || req.Body == nil {
		return fmt.Errorf("invalid request")
	}
	return decodeJSON(req.Body, obj)
}

func (jsonBinding) BindBody(body []byte, obj interface{}) error {
	return decodeJSON(bytes.NewReader(body), obj)
}

func decodeJSON(r io.Reader, obj interface{}) error {
	decoder := json.NewDecoder(r)
	if EnableDecoderUseNumber {
		decoder.UseNumber()
	}
	if err := decoder.Decode(obj); err != nil {
		return err
	}
	return validate(obj)
}

func validate(obj interface{}) error {
	if binding.Validator == nil {
		return nil
	}
	return binding.Validator.ValidateStruct(obj)
}
