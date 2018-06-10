package client

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"git.xiagaogao.com/coffee/boot/errors"
	"go.uber.org/zap"
)

func NewHTTPRequest(method, urlStr string,errorService errors.Service,logger *zap.Logger) (HTTPRequest, error) {
	req, err := http.NewRequest(method, urlStr, nil)
	if err != nil {
		return nil, err
	}
	return &httpRequestImpl{
		req: req,
		errorService:errorService,
		logger:logger,
	}, nil
}

type httpRequestImpl struct {
	req       *http.Request
	cookieJar http.CookieJar
	transport *http.Transport
	errorService errors.Service
	logger *zap.Logger
}

func (impl *httpRequestImpl) SetTransport(transport *http.Transport) {
	impl.transport = transport
}

func (impl *httpRequestImpl) SetMethod(method string) {
	impl.req.Method = method
}
func (impl *httpRequestImpl) SetHeader(k, v string) {
	impl.req.Header.Set(k, v)
}
func (impl *httpRequestImpl) SetCookieJar(cookieJar http.CookieJar) {
	impl.cookieJar = cookieJar
}
func (impl *httpRequestImpl) SetBody(body []byte) {
	impl.req.Body = ioutil.NopCloser(bytes.NewReader(body))
}
func (impl *httpRequestImpl) SetBodyStream(reader io.ReadCloser) {
	impl.req.Body = reader
}

func (impl *httpRequestImpl) SetURI(requestURL string) error {
	_url, err := url.ParseRequestURI(requestURL)
	if err != nil {
		return err
	}
	impl.req.URL = _url
	return nil
}
func (impl *httpRequestImpl) SetBasicAuth(username, password string) {
	impl.req.SetBasicAuth(username, password)
}
func (impl *httpRequestImpl) SetContentType(contentType string) {
	impl.req.Header.Set("Content-Type", contentType)
}
func (impl *httpRequestImpl) SetCookie(cookie *http.Cookie) {
	impl.req.AddCookie(cookie)
}
func (impl *httpRequestImpl) SetReferer(referer string) {
	impl.req.Header.Set("referer", referer)
}
func (impl *httpRequestImpl) SetUserAgent(userAgent string) {
	impl.req.Header.Set("user-agent", userAgent)
}

func (impl *httpRequestImpl) SetProto(proto string) {
	impl.req.Proto = proto
}
func (impl *httpRequestImpl) GetRealRequest() *http.Request {
	return impl.req
}
