package client

import (
	"io"
	"net/http"
)

func newResponse(resp *http.Response) Response {
	return &_Response{
		resp: resp,
	}
}

type _Response struct {
	resp *http.Response
}

func (r *_Response) GetBody() io.ReadCloser {
	return r.resp.Body
}
func (r *_Response) GetRealResponse() *http.Response {
	return r.resp
}

func (r *_Response) GetHeader() http.Header {
	return r.resp.Header
}

func (r *_Response) GetContentType() string {
	return r.resp.Header.Get("Content-Type")
}
