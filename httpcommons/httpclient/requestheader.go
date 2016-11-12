package httpclient

import (
	"github.com/valyala/fasthttp"
	"bufio"
	"io"
)

type _HttpRequestHeader struct {
	header *fasthttp.RequestHeader
}

func (this *_HttpRequestHeader)getFastHttpRequestHeader() *fasthttp.RequestHeader {
	return this.header
}
func (this *_HttpRequestHeader)SetByteRange(startPos, endPos int) {
	this.header.SetByteRange(startPos, endPos)
}
func (this *_HttpRequestHeader)ConnectionClose() bool {
	return this.header.ConnectionClose()
}
func (this *_HttpRequestHeader)SetConnectionClose() {
	this.header.SetConnectionClose()
}
func (this *_HttpRequestHeader)ResetConnectionClose() {
	this.header.ResetConnectionClose()
}
func (this *_HttpRequestHeader)ConnectionUpgrade() bool {
	return this.header.ConnectionUpgrade()
}
func (this *_HttpRequestHeader)ContentLength() int {
	return this.header.ContentLength()
}
func (this *_HttpRequestHeader)SetContentLength(contentLength int) {
	this.header.SetContentLength(contentLength)
}
func (this *_HttpRequestHeader)ContentType() []byte {
	return this.header.ContentType()
}
func (this *_HttpRequestHeader)SetContentType(contentType string) {
	this.header.SetContentType(contentType)
}
func (this *_HttpRequestHeader)SetContentTypeBytes(contentType []byte) {
	this.header.SetContentTypeBytes(contentType)
}
func (this *_HttpRequestHeader)SetMultipartFormBoundary(boundary string) {
	this.header.SetMultipartFormBoundary(boundary)
}
func (this *_HttpRequestHeader)SetMultipartFormBoundaryBytes(boundary []byte) {
	this.header.SetMultipartFormBoundaryBytes(boundary)
}
func (this *_HttpRequestHeader)MultipartFormBoundary() []byte {
	return this.header.MultipartFormBoundary()
}
func (this *_HttpRequestHeader)Host() []byte {
	return this.header.Host()
}
func (this *_HttpRequestHeader)SetHost(host string) {
	this.header.SetHost(host)
}
func (this *_HttpRequestHeader)SetHostBytes(host []byte) {
	this.header.SetHostBytes(host)
}
func (this *_HttpRequestHeader)UserAgent() []byte {
	return this.header.UserAgent()
}
func (this *_HttpRequestHeader)SetUserAgent(userAgent string) {
	this.header.SetUserAgent(userAgent)
}
func (this *_HttpRequestHeader)SetUserAgentBytes(userAgent []byte) {
	this.header.SetUserAgentBytes(userAgent)
}
func (this *_HttpRequestHeader)Referer() []byte {
	return this.header.Referer()
}
func (this *_HttpRequestHeader)SetReferer(referer string) {
	this.header.SetReferer(referer)
}
func (this *_HttpRequestHeader)SetRefererBytes(referer []byte) {
	this.header.SetRefererBytes(referer)
}
func (this *_HttpRequestHeader)Method() []byte {
	return this.header.Method()
}
func (this *_HttpRequestHeader)SetMethod(method string) {
	this.header.SetMethod(method)
}
func (this *_HttpRequestHeader)SetMethodBytes(method []byte) {
	this.header.SetMethodBytes(method)
}
func (this *_HttpRequestHeader)RequestURI() []byte {
	return this.header.RequestURI()
}
func (this *_HttpRequestHeader)SetRequestURI(requestURI string) {
	this.header.SetRequestURI(requestURI)
}
func (this *_HttpRequestHeader)SetRequestURIBytes(requestURI []byte) {
	this.header.SetRequestURIBytes(requestURI)
}
func (this *_HttpRequestHeader)IsGet() bool {
	return this.header.IsGet()
}
func (this *_HttpRequestHeader)IsPost() bool {
	return this.header.IsPost()
}
func (this *_HttpRequestHeader)IsPut() bool {
	return this.header.IsPut()
}
func (this *_HttpRequestHeader)IsHead() bool {
	return this.header.IsHead()
}
func (this *_HttpRequestHeader)IsDelete() bool {
	return this.header.IsDelete()
}
func (this *_HttpRequestHeader)IsHTTP11() bool {
	return this.header.IsHTTP11()
}
func (this *_HttpRequestHeader)HasAcceptEncoding(acceptEncoding string) bool {
	return this.header.HasAcceptEncoding(acceptEncoding)
}
func (this *_HttpRequestHeader)HasAcceptEncodingBytes(acceptEncoding []byte) bool {
	return this.header.HasAcceptEncodingBytes(acceptEncoding)
}
func (this *_HttpRequestHeader)Len() int {
	return this.header.Len()
}
func (this *_HttpRequestHeader)DisableNormalizing() {
	this.header.DisableNormalizing()
}
func (this *_HttpRequestHeader)Reset() {
	this.header.Reset()
}
func (this *_HttpRequestHeader)CopyTo(dst HttpRequestHeader) {
	this.header.CopyTo(dst.getFastHttpRequestHeader())
}
func (this *_HttpRequestHeader)VisitAll(f func(key, value []byte)) {
	this.header.VisitAll(f)
}
func (this *_HttpRequestHeader)VisitAllCookie(f func(key, value []byte)) {
	this.header.VisitAllCookie(f)
}
func (this *_HttpRequestHeader)Del(key string) {
	this.header.Del(key)
}
func (this *_HttpRequestHeader)DelBytes(key []byte) {
	this.header.DelBytes(key)
}
func (this *_HttpRequestHeader)Add(key, value string) {
	this.header.Add(key, value)
}
func (this *_HttpRequestHeader)AddBytesK(key []byte, value string) {
	this.header.SetBytesK(key, value)
}
func (this *_HttpRequestHeader)AddBytesV(key string, value []byte) {
	this.header.AddBytesV(key, value)
}
func (this *_HttpRequestHeader)AddBytesKV(key, value []byte) {
	this.header.AddBytesKV(key, value)
}
func (this *_HttpRequestHeader)SetBytesK(key []byte, value string) {
	this.header.AddBytesK(key, value)
}
func (this *_HttpRequestHeader)SetBytesV(key string, value []byte) {
	this.header.SetBytesV(key, value)
}
func (this *_HttpRequestHeader)SetBytesKV(key, value []byte) {
	this.header.SetBytesKV(key, value)
}
func (this *_HttpRequestHeader)SetCanonical(key, value []byte) {
	this.header.SetCanonical(key, value)
}
func (this *_HttpRequestHeader)SetCookie(key string,value string) {
	this.header.SetCookie(key,value)
}
func (this *_HttpRequestHeader)SetCookieBytesK(key []byte, value string) {
	this.header.SetCookieBytesK(key, value)
}
func (this *_HttpRequestHeader)SetCookieBytesKV(key []byte, value []byte) {
	this.header.SetCookieBytesKV(key, value)
}
func (this *_HttpRequestHeader)Peek(key string) []byte {
	return this.header.Peek(key)
}
func (this *_HttpRequestHeader)PeekBytes(key []byte) []byte {
	return this.header.PeekBytes(key)
}
func (this *_HttpRequestHeader)Cookie(key string) []byte {
	return this.header.Cookie(key)
}
func (this *_HttpRequestHeader)CookieBytes(key []byte) []byte {
	return this.header.CookieBytes(key)
}
func (this *_HttpRequestHeader)Read(r *bufio.Reader) error {
	return this.header.Read(r)
}
func (this *_HttpRequestHeader)Write(w *bufio.Writer) error {
	return this.header.Write(w)
}
func (this *_HttpRequestHeader)WriteTo(w io.Writer) (int64, error) {
	return this.header.WriteTo(w)
}
func (this *_HttpRequestHeader)Header() []byte {
	return this.header.Header()
}
func (this *_HttpRequestHeader)String() string {
	return this.header.String()
}
func (this *_HttpRequestHeader)AppendBytes(dst []byte) []byte {
	return this.header.AppendBytes(dst)
}

func (this *_HttpRequestHeader)DelCookieBytes(key []byte) {
	this.header.DelCookieBytes(key)
}
func (this *_HttpRequestHeader)DelCookie(key string) {
	this.header.DelCookie(key)
}
func (this *_HttpRequestHeader)DelAllCookies() {
	this.header.DelAllCookies()
}
