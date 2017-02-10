package client

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

func NewRequest() Request {
	return &_Request{
		req: new(http.Request),
	}
}

type _Request struct {
	req       *http.Request
	cookieJar http.CookieJar
	reader    io.ReadCloser
}

func (_req *_Request) SeMethod(method string) {
	_req.req.Method = method
}
func (_req *_Request) SetHeader(k, v string) {
	_req.req.Header.Set(k, v)
}
func (_req *_Request) SetCookieJar(cookieJar http.CookieJar) {
	_req.cookieJar = cookieJar
}
func (_req *_Request) SetBody(body []byte) {
	_req.reader = ioutil.NopCloser(bytes.NewReader(body))
}
func (_req *_Request) SetBodyStream(reader io.ReadCloser) {
	_req.reader = reader
}

func (_req *_Request) SetURI(requestURL string) error {
	_url, err := url.ParseRequestURI(requestURL)
	if err != nil {
		return err
	}
	_req.req.URL = _url
	return nil
}
func (_req *_Request) SetBasicAuth(username, password string) {
	_req.req.SetBasicAuth(username, password)
}
func (_req *_Request) SetContentType(contentType string) {
	_req.req.Header.Set("Content-Type", contentType)
}
func (_req *_Request) SetCookie(cookie *http.Cookie) {
	_req.req.AddCookie(cookie)
}
func (_req *_Request) SetReferer(referer string) {
	_req.req.Header.Set("referer", referer)
}
func (_req *_Request) SetUserAgent(userAgent string) {
	_req.req.Header.Set("user-agent", userAgent)
}

func (_req *_Request) SetProto(proto string) {
	_req.req.Proto = proto
}
func (_req *_Request) GetRealRequest() *http.Request {
	return _req.req
}
