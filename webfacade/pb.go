package webfacade

import (
	"github.com/coffeehc/commons/coder"
	"github.com/gogo/protobuf/proto"
	"net/http"
)

var PBContentType = "application/x-protobuf"

type ProtobufRender struct {
	Data proto.Message
}

// Render (JSON) writes data with custom ContentType.
func (r ProtobufRender) Render(w http.ResponseWriter) (err error) {
	if err = writeProtobuf(w, r.Data); err != nil {
		panic(err)
	}
	return
}

// WriteContentType (JSON) writes JSON ContentType.
func (r ProtobufRender) WriteContentType(w http.ResponseWriter) {
	WriteContentType(w, []string{PBContentType})
}

// WriteJSON marshals the given interface object and writes it with custom ContentType.
func writeProtobuf(w http.ResponseWriter, obj proto.Message) error {
	WriteContentType(w, []string{PBContentType})
	pbBytes, err := coder.PBCoder.Marshal(obj)
	if err != nil {
		return err
	}
	_, err = w.Write(pbBytes)
	return err
}
