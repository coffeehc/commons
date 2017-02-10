package client

import (
	"io"
	"net/http"
)

const (
	ContentTypeJson   = "application/json"
	ContentTypeStream = "application/octet-stream"
)

type HTTPClient interface {
	Get(url string) (HTTPResponse, error)
	POST(url string, body io.Reader, contentType string) (HTTPResponse, error)
	Do(HTTPRequest) (HTTPResponse, error)
}

type HTTPRequest interface {
	SeMethod(method string)
	SetHeader(k, v string)
	SetCookieJar(http.CookieJar)
	SetBody(body []byte)
	SetBodyStream(reader io.ReadCloser)
	SetURI(uri string) error
	SetBasicAuth(username, password string)
	SetContentType(contentType string)
	SetCookie(cookie *http.Cookie)
	SetProto(proto string) // HTTP/1.0  HTTP/1.1  HTTP/2 默认使用HTTP/1.1
	SetReferer(referer string)
	SetUserAgent(userAgent string)

	GetRealRequest() *http.Request

	//build Client Options
	//UseProxy(proxyIP string, scheme string)
}

type HTTPResponse interface {
	GetBody() io.ReadCloser
	GetRealResponse() *http.Response
	GetHeader() http.Header
	GetContentType() string
}

type CookieJarManager interface {
	GetCookieJar(key string) http.CookieJar
}
