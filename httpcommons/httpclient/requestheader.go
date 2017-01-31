package httpclient

import (
	"bufio"
	"io"

	"github.com/valyala/fasthttp"
)

type _RequestHeader struct {
	header *fasthttp.RequestHeader
}

func (requestHeader *_RequestHeader) getFastHTTPRequestHeader() *fasthttp.RequestHeader {
	return requestHeader.header
}
func (requestHeader *_RequestHeader) SetByteRange(startPos, endPos int) {
	requestHeader.header.SetByteRange(startPos, endPos)
}
func (requestHeader *_RequestHeader) ConnectionClose() bool {
	return requestHeader.header.ConnectionClose()
}
func (requestHeader *_RequestHeader) SetConnectionClose() {
	requestHeader.header.SetConnectionClose()
}
func (requestHeader *_RequestHeader) ResetConnectionClose() {
	requestHeader.header.ResetConnectionClose()
}
func (requestHeader *_RequestHeader) ConnectionUpgrade() bool {
	return requestHeader.header.ConnectionUpgrade()
}
func (requestHeader *_RequestHeader) ContentLength() int {
	return requestHeader.header.ContentLength()
}
func (requestHeader *_RequestHeader) SetContentLength(contentLength int) {
	requestHeader.header.SetContentLength(contentLength)
}
func (requestHeader *_RequestHeader) ContentType() []byte {
	return requestHeader.header.ContentType()
}
func (requestHeader *_RequestHeader) SetContentType(contentType string) {
	requestHeader.header.SetContentType(contentType)
}
func (requestHeader *_RequestHeader) SetContentTypeBytes(contentType []byte) {
	requestHeader.header.SetContentTypeBytes(contentType)
}
func (requestHeader *_RequestHeader) SetMultipartFormBoundary(boundary string) {
	requestHeader.header.SetMultipartFormBoundary(boundary)
}
func (requestHeader *_RequestHeader) SetMultipartFormBoundaryBytes(boundary []byte) {
	requestHeader.header.SetMultipartFormBoundaryBytes(boundary)
}
func (requestHeader *_RequestHeader) MultipartFormBoundary() []byte {
	return requestHeader.header.MultipartFormBoundary()
}
func (requestHeader *_RequestHeader) Host() []byte {
	return requestHeader.header.Host()
}
func (requestHeader *_RequestHeader) SetHost(host string) {
	requestHeader.header.SetHost(host)
}
func (requestHeader *_RequestHeader) SetHostBytes(host []byte) {
	requestHeader.header.SetHostBytes(host)
}
func (requestHeader *_RequestHeader) UserAgent() []byte {
	return requestHeader.header.UserAgent()
}
func (requestHeader *_RequestHeader) SetUserAgent(userAgent string) {
	requestHeader.header.SetUserAgent(userAgent)
}
func (requestHeader *_RequestHeader) SetUserAgentBytes(userAgent []byte) {
	requestHeader.header.SetUserAgentBytes(userAgent)
}
func (requestHeader *_RequestHeader) Referer() []byte {
	return requestHeader.header.Referer()
}
func (requestHeader *_RequestHeader) SetReferer(referer string) {
	requestHeader.header.SetReferer(referer)
}
func (requestHeader *_RequestHeader) SetRefererBytes(referer []byte) {
	requestHeader.header.SetRefererBytes(referer)
}
func (requestHeader *_RequestHeader) Method() []byte {
	return requestHeader.header.Method()
}
func (requestHeader *_RequestHeader) SetMethod(method string) {
	requestHeader.header.SetMethod(method)
}
func (requestHeader *_RequestHeader) SetMethodBytes(method []byte) {
	requestHeader.header.SetMethodBytes(method)
}
func (requestHeader *_RequestHeader) RequestURI() []byte {
	return requestHeader.header.RequestURI()
}
func (requestHeader *_RequestHeader) SetRequestURI(requestURI string) {
	requestHeader.header.SetRequestURI(requestURI)
}
func (requestHeader *_RequestHeader) SetRequestURIBytes(requestURI []byte) {
	requestHeader.header.SetRequestURIBytes(requestURI)
}
func (requestHeader *_RequestHeader) IsGet() bool {
	return requestHeader.header.IsGet()
}
func (requestHeader *_RequestHeader) IsPost() bool {
	return requestHeader.header.IsPost()
}
func (requestHeader *_RequestHeader) IsPut() bool {
	return requestHeader.header.IsPut()
}
func (requestHeader *_RequestHeader) IsHead() bool {
	return requestHeader.header.IsHead()
}
func (requestHeader *_RequestHeader) IsDelete() bool {
	return requestHeader.header.IsDelete()
}
func (requestHeader *_RequestHeader) IsHTTP11() bool {
	return requestHeader.header.IsHTTP11()
}
func (requestHeader *_RequestHeader) HasAcceptEncoding(acceptEncoding string) bool {
	return requestHeader.header.HasAcceptEncoding(acceptEncoding)
}
func (requestHeader *_RequestHeader) HasAcceptEncodingBytes(acceptEncoding []byte) bool {
	return requestHeader.header.HasAcceptEncodingBytes(acceptEncoding)
}
func (requestHeader *_RequestHeader) Len() int {
	return requestHeader.header.Len()
}
func (requestHeader *_RequestHeader) DisableNormalizing() {
	requestHeader.header.DisableNormalizing()
}
func (requestHeader *_RequestHeader) Reset() {
	requestHeader.header.Reset()
}
func (requestHeader *_RequestHeader) CopyTo(dst RequestHeader) {
	requestHeader.header.CopyTo(dst.getFastHTTPRequestHeader())
}
func (requestHeader *_RequestHeader) VisitAll(f func(key, value []byte)) {
	requestHeader.header.VisitAll(f)
}
func (requestHeader *_RequestHeader) VisitAllCookie(f func(key, value []byte)) {
	requestHeader.header.VisitAllCookie(f)
}
func (requestHeader *_RequestHeader) Del(key string) {
	requestHeader.header.Del(key)
}
func (requestHeader *_RequestHeader) DelBytes(key []byte) {
	requestHeader.header.DelBytes(key)
}
func (requestHeader *_RequestHeader) Add(key, value string) {
	requestHeader.header.Add(key, value)
}
func (requestHeader *_RequestHeader) AddBytesK(key []byte, value string) {
	requestHeader.header.SetBytesK(key, value)
}
func (requestHeader *_RequestHeader) AddBytesV(key string, value []byte) {
	requestHeader.header.AddBytesV(key, value)
}
func (requestHeader *_RequestHeader) AddBytesKV(key, value []byte) {
	requestHeader.header.AddBytesKV(key, value)
}
func (requestHeader *_RequestHeader) SetBytesK(key []byte, value string) {
	requestHeader.header.AddBytesK(key, value)
}
func (requestHeader *_RequestHeader) SetBytesV(key string, value []byte) {
	requestHeader.header.SetBytesV(key, value)
}
func (requestHeader *_RequestHeader) SetBytesKV(key, value []byte) {
	requestHeader.header.SetBytesKV(key, value)
}
func (requestHeader *_RequestHeader) SetCanonical(key, value []byte) {
	requestHeader.header.SetCanonical(key, value)
}
func (requestHeader *_RequestHeader) SetCookie(key string, value string) {
	requestHeader.header.SetCookie(key, value)
}
func (requestHeader *_RequestHeader) SetCookieBytesK(key []byte, value string) {
	requestHeader.header.SetCookieBytesK(key, value)
}
func (requestHeader *_RequestHeader) SetCookieBytesKV(key []byte, value []byte) {
	requestHeader.header.SetCookieBytesKV(key, value)
}
func (requestHeader *_RequestHeader) Peek(key string) []byte {
	return requestHeader.header.Peek(key)
}
func (requestHeader *_RequestHeader) PeekBytes(key []byte) []byte {
	return requestHeader.header.PeekBytes(key)
}
func (requestHeader *_RequestHeader) Cookie(key string) []byte {
	return requestHeader.header.Cookie(key)
}
func (requestHeader *_RequestHeader) CookieBytes(key []byte) []byte {
	return requestHeader.header.CookieBytes(key)
}
func (requestHeader *_RequestHeader) Read(r *bufio.Reader) error {
	return requestHeader.header.Read(r)
}
func (requestHeader *_RequestHeader) Write(w *bufio.Writer) error {
	return requestHeader.header.Write(w)
}
func (requestHeader *_RequestHeader) WriteTo(w io.Writer) (int64, error) {
	return requestHeader.header.WriteTo(w)
}
func (requestHeader *_RequestHeader) Header() []byte {
	return requestHeader.header.Header()
}
func (requestHeader *_RequestHeader) String() string {
	return requestHeader.header.String()
}
func (requestHeader *_RequestHeader) AppendBytes(dst []byte) []byte {
	return requestHeader.header.AppendBytes(dst)
}

func (requestHeader *_RequestHeader) DelCookieBytes(key []byte) {
	requestHeader.header.DelCookieBytes(key)
}
func (requestHeader *_RequestHeader) DelCookie(key string) {
	requestHeader.header.DelCookie(key)
}
func (requestHeader *_RequestHeader) DelAllCookies() {
	requestHeader.header.DelAllCookies()
}
