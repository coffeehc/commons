package httpclient

import (
	"io"
	"mime/multipart"
	"bufio"
	"time"
	"github.com/valyala/fasthttp"
)

type StreamWriter func(w *bufio.Writer)

type HttpArgs interface {
	getFastHttpAtgs() *fasthttp.Args

	Reset()
	CopyTo(dst HttpArgs)
	VisitAll(f func(key, value []byte))
	Len() int
	Parse(s string)
	ParseBytes(b []byte)
	String() string
	QueryString() []byte
	AppendBytes(dst []byte) []byte
	WriteTo(w io.Writer) (int64, error)
	Del(key string)
	DelBytes(key []byte)
	Add(key, value string)
	AddBytesK(key []byte, value string)
	AddBytesV(key string, value []byte)
	AddBytesKV(key, value []byte)
	Set(key, value string)
	SetBytesK(key []byte, value string)
	SetBytesV(key string, value []byte)
	SetBytesKV(key, value []byte)
	Peek(key string) []byte
	PeekBytes(key []byte) []byte
	PeekMulti(key string) [][]byte
	PeekMultiBytes(key []byte) [][]byte
	Has(key string) bool
	HasBytes(key []byte) bool
	GetUint(key string) (int, error)
	SetUint(key string, value int)
	SetUintBytes(key []byte, value int)
	GetUintOrZero(key string) int
	GetUfloat(key string) (float64, error)
	GetUfloatOrZero(key string) float64
}

type HttpCookie interface {
	getFastHttpCookie() *fasthttp.Cookie

	CopyTo(src HttpCookie)
	HTTPOnly() bool
	SetHTTPOnly(httpOnly bool)
	Secure() bool
	SetSecure(secure bool)
	Path() []byte
	SetPath(path string)
	SetPathBytes(path []byte)
	Domain() []byte
	SetDomain(domain string)
	SetDomainBytes(domain []byte)
	Expire() time.Time
	SetExpire(expire time.Time)
	Value() []byte
	SetValue(value string)
	SetValueBytes(value []byte)
	Key() []byte
	SetKey(key string)
	SetKeyBytes(key []byte)
	Reset()
	AppendBytes(dst []byte) []byte
	Cookie() []byte
	String() string
	WriteTo(w io.Writer) (int64, error)
	Parse(src string) error
	ParseBytes(src []byte) error
}

type HttpUri interface {
	getFastHttpUri() *fasthttp.URI

	CopyTo(dst HttpUri)
	Hash() []byte
	SetHash(hash string)
	SetHashBytes(hash []byte)
	QueryString() []byte
	SetQueryString(queryString string)
	SetQueryStringBytes(queryString []byte)
	Path() []byte
	SetPath(path string)
	SetPathBytes(path []byte)
	PathOriginal() []byte
	Scheme() []byte
	SetScheme(scheme string)
	SetSchemeBytes(scheme []byte)
	Reset()
	Host() []byte
	SetHost(host string)
	SetHostBytes(host []byte)
	Parse(host, uri []byte)
	RequestURI() []byte
	LastPathSegment() []byte
	Update(newURI string)
	UpdateBytes(newURI []byte)
	FullURI() []byte
	AppendBytes(dst []byte) []byte
	WriteTo(w io.Writer) (int64, error)
	QueryArgs() HttpArgs
}

type HttpRequestHeader interface {
	getFastHttpRequestHeader() *fasthttp.RequestHeader

	SetByteRange(startPos, endPos int)
	ConnectionClose() bool
	SetConnectionClose()
	ResetConnectionClose()
	ConnectionUpgrade() bool
	ContentLength() int
	SetContentLength(contentLength int)
	ContentType() []byte
	SetContentType(contentType string)
	SetContentTypeBytes(contentType []byte)
	SetMultipartFormBoundary(boundary string)
	SetMultipartFormBoundaryBytes(boundary []byte)
	MultipartFormBoundary() []byte
	Host() []byte
	SetHost(host string)
	SetHostBytes(host []byte)
	UserAgent() []byte
	SetUserAgent(userAgent string)
	SetUserAgentBytes(userAgent []byte)
	Referer() []byte
	SetReferer(referer string)
	SetRefererBytes(referer []byte)
	Method() []byte
	SetMethod(method string)
	SetMethodBytes(method []byte)
	RequestURI() []byte
	SetRequestURI(requestURI string)
	SetRequestURIBytes(requestURI []byte)
	IsGet() bool
	IsPost() bool
	IsPut() bool
	IsHead() bool
	IsDelete() bool
	IsHTTP11() bool
	HasAcceptEncoding(acceptEncoding string) bool
	HasAcceptEncodingBytes(acceptEncoding []byte) bool
	Len() int
	DisableNormalizing()
	Reset()
	CopyTo(dst HttpRequestHeader)
	VisitAll(f func(key, value []byte))
	VisitAllCookie(f func(key, value []byte))
	Del(key string)
	DelBytes(key []byte)
	SetCookie(cookie HttpCookie)
	SetCookieBytesK(key []byte, value string)
	SetCookieBytesKV(key []byte,value []byte)
	DelCookieBytes(key []byte)
	DelCookie(key string)
	DelAllCookies()
	Add(key, value string)
	AddBytesK(key []byte, value string)
	AddBytesV(key string, value []byte)
	AddBytesKV(key, value []byte)
	SetBytesK(key []byte, value string)
	SetBytesV(key string, value []byte)
	SetBytesKV(key, value []byte)
	SetCanonical(key, value []byte)
	Peek(key string) []byte
	PeekBytes(key []byte) []byte
	Cookie(key string) []byte
	CookieBytes(key []byte) []byte
	Read(r *bufio.Reader) error
	Write(w *bufio.Writer) error
	WriteTo(w io.Writer) (int64, error)
	Header() []byte
	String() string
	AppendBytes(dst []byte) []byte

}

type HttpResponseHeader interface {
	getFastHttpResponseHeader() *fasthttp.ResponseHeader

	SetContentRange(startPos, endPos, contentLength int)
	StatusCode() int
	SetStatusCode(statusCode int)
	SetLastModified(t time.Time)
	ConnectionClose() bool
	SetConnectionClose()
	ResetConnectionClose()
	ConnectionUpgrade() bool
	ContentLength() int
	SetContentLength(contentLength int)
	ContentType() []byte
	SetContentType(contentType string)
	SetContentTypeBytes(contentType []byte)
	Server() []byte
	SetServer(server string)
	SetServerBytes(server []byte)
	IsHTTP11() bool
	Len() int
	DisableNormalizing()
	Reset()
	CopyTo(dst HttpResponseHeader)
	VisitAll(f func(key, value []byte))
	VisitAllCookie(f func(key, value []byte))
	Del(key string)
	DelBytes(key []byte)
	Add(key, value string)
	AddBytesK(key []byte, value string)
	AddBytesV(key string, value []byte)
	AddBytesKV(key, value []byte)
	Set(key, value string)
	SetBytesK(key []byte, value string)
	SetBytesV(key string, value []byte)
	SetBytesKV(key, value []byte)
	SetCanonical(key, value []byte)
	SetCookie(cookie HttpCookie)
	DelClientCookie(key string)
	DelClientCookieBytes(key []byte)
	DelCookieBytes(key []byte)
	DelCookie(key string)
	DelAllCookies()
	Peek(key string) []byte
	PeekBytes(key []byte) []byte
	Cookie(key string) []byte
	Read(r *bufio.Reader) error
	Write(w *bufio.Writer) error
	WriteTo(w io.Writer) (int64, error)
	Header() []byte
	String() string
	AppendBytes(dst []byte) []byte
}


type HttpRequest interface {
	getFastHttpRequest() *fasthttp.Request

	SetHost(host string)
	SetHostBytes(host []byte)
	Host() []byte
	GetRequestURI(requestURI string)
	SetRequestURI(requestURI string)
	SetRequestURIBytes(requestURI []byte)
	RequestURI() []byte
	ConnectionClose() bool
	SetConnectionClose()
	SendFile(path string) error
	SetBodyStream(bodyStream io.Reader, bodySize int)
	SetBodyStreamWriter(sw StreamWriter)
	BodyWriter() io.Writer
	BodyGunzip() ([]byte, error)
	BodyInflate() ([]byte, error)
	BodyWriteTo(w io.Writer) error
	ReleaseBody(size int)
	SwapBody(body []byte) []byte
	Body() []byte
	AppendBody(p []byte)
	AppendBodyString(s string)
	SetBody(body []byte)
	SetBodyString(body string)
	ResetBody()
	CopyTo(dst *HttpRequest)
	URI() HttpUri
	PostArgs() HttpArgs
	MultipartForm() (*multipart.Form, error)
	Reset()
	RemoveMultipartFormFiles()
	Read(r *bufio.Reader) error
	ReadLimitBody(r *bufio.Reader, maxBodySize int) error
	MayContinue() bool
	ContinueReadBody(r *bufio.Reader, maxBodySize int) error
	WriteTo(w io.Writer) (int64, error)
	Write(w *bufio.Writer) error
	String() string
}

type HttpResponse interface {
	getFastHttpResponse() *fasthttp.Response

	StatusCode() int
	SetStatusCode(statusCode int)
	ConnectionClose() bool
	SetConnectionClose()
	SendFile(path string) error
	SetBodyStream(bodyStream io.Reader, bodySize int)
	IsBodyStream() bool
	SetBodyStreamWriter(sw StreamWriter)
	BodyWriter() io.Writer
	Body() []byte
	BodyGunzip() ([]byte, error)
	BodyInflate() ([]byte, error)
	BodyWriteTo(w io.Writer) error
	AppendBody(p []byte)
	AppendBodyString(s string)
	SetBody(body []byte)
	SetBodyString(body string)
	ResetBody()
	ReleaseBody(size int)
	SwapBody(body []byte) []byte
	CopyTo(dst HttpResponse)
	Reset()
	Read(r *bufio.Reader) error
	ReadLimitBody(r *bufio.Reader, maxBodySize int) error
	WriteTo(w io.Writer) (int64, error)
	WriteGzip(w *bufio.Writer) error
	WriteGzipLevel(w *bufio.Writer, level int) error
	WriteDeflate(w *bufio.Writer) error
	WriteDeflateLevel(w *bufio.Writer, level int) error
	Write(w *bufio.Writer) error
	String() string
}

type HttpClient interface {
	Get(dst []byte, url string) (statusCode int, body []byte, err error)
	GetTimeout(dst []byte, url string, timeout time.Duration) (statusCode int, body []byte, err error)
	GetDeadline(dst []byte, url string, deadline time.Time) (statusCode int, body []byte, err error)
	Post(dst []byte, url string, postArgs HttpArgs) (statusCode int, body []byte, err error)
	DoTimeout(req HttpRequest, resp HttpResponse, timeout time.Duration) error
	DoDeadline(req HttpRequest, resp HttpResponse, deadline time.Time) error
	Do(req HttpRequest, resp HttpResponse) error
}


