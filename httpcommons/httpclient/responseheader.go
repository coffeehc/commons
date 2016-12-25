package httpclient

import (
	"bufio"
	"github.com/valyala/fasthttp"
	"io"
	"time"
)

type _ResponseHeader struct {
	headre *fasthttp.ResponseHeader
}

func (responseHeader *_ResponseHeader) getFastHTTPResponseHeader() *fasthttp.ResponseHeader {
	return responseHeader.headre
}

func (responseHeader *_ResponseHeader) SetContentRange(startPos, endPos, contentLength int) {
	responseHeader.headre.SetContentRange(startPos, endPos, contentLength)
}
func (responseHeader *_ResponseHeader) StatusCode() int {
	return responseHeader.headre.StatusCode()
}
func (responseHeader *_ResponseHeader) SetStatusCode(statusCode int) {
	responseHeader.headre.SetStatusCode(statusCode)
}
func (responseHeader *_ResponseHeader) SetLastModified(t time.Time) {
	responseHeader.headre.SetLastModified(t)
}
func (responseHeader *_ResponseHeader) ConnectionClose() bool {
	return responseHeader.headre.ConnectionClose()
}
func (responseHeader *_ResponseHeader) SetConnectionClose() {
	responseHeader.headre.SetConnectionClose()
}
func (responseHeader *_ResponseHeader) ResetConnectionClose() {
	responseHeader.headre.ResetConnectionClose()
}
func (responseHeader *_ResponseHeader) ConnectionUpgrade() bool {
	return responseHeader.headre.ConnectionUpgrade()
}
func (responseHeader *_ResponseHeader) ContentLength() int {
	return responseHeader.headre.ContentLength()
}
func (responseHeader *_ResponseHeader) SetContentLength(contentLength int) {
	responseHeader.headre.SetContentLength(contentLength)
}
func (responseHeader *_ResponseHeader) ContentType() []byte {
	return responseHeader.headre.ContentType()
}
func (responseHeader *_ResponseHeader) SetContentType(contentType string) {
	responseHeader.headre.SetContentType(contentType)
}
func (responseHeader *_ResponseHeader) SetContentTypeBytes(contentType []byte) {
	responseHeader.headre.SetContentTypeBytes(contentType)
}
func (responseHeader *_ResponseHeader) Server() []byte {
	return responseHeader.headre.Server()
}
func (responseHeader *_ResponseHeader) SetServer(server string) {
	responseHeader.headre.SetServer(server)
}
func (responseHeader *_ResponseHeader) SetServerBytes(server []byte) {
	responseHeader.headre.SetServerBytes(server)
}
func (responseHeader *_ResponseHeader) IsHTTP11() bool {
	return responseHeader.headre.IsHTTP11()
}
func (responseHeader *_ResponseHeader) Len() int {
	return responseHeader.headre.Len()
}
func (responseHeader *_ResponseHeader) DisableNormalizing() {
	responseHeader.headre.DisableNormalizing()
}
func (responseHeader *_ResponseHeader) Reset() {
	responseHeader.headre.Reset()
}
func (responseHeader *_ResponseHeader) CopyTo(dst ResponseHeader) {
	responseHeader.headre.CopyTo(dst.getFastHTTPResponseHeader())
}
func (responseHeader *_ResponseHeader) VisitAll(f func(key, value []byte)) {
	responseHeader.headre.VisitAll(f)
}
func (responseHeader *_ResponseHeader) VisitAllCookie(f func(key, value []byte)) {
	responseHeader.headre.VisitAllCookie(f)
}
func (responseHeader *_ResponseHeader) Del(key string) {
	responseHeader.headre.Del(key)
}
func (responseHeader *_ResponseHeader) DelBytes(key []byte) {
	responseHeader.headre.DelBytes(key)
}
func (responseHeader *_ResponseHeader) Add(key, value string) {
	responseHeader.headre.Add(key, value)
}
func (responseHeader *_ResponseHeader) AddBytesK(key []byte, value string) {
	responseHeader.headre.AddBytesK(key, value)
}
func (responseHeader *_ResponseHeader) AddBytesV(key string, value []byte) {
	responseHeader.headre.AddBytesV(key, value)
}
func (responseHeader *_ResponseHeader) AddBytesKV(key, value []byte) {
	responseHeader.headre.AddBytesKV(key, value)
}
func (responseHeader *_ResponseHeader) Set(key, value string) {
	responseHeader.headre.Set(key, value)
}
func (responseHeader *_ResponseHeader) SetBytesK(key []byte, value string) {
	responseHeader.headre.SetBytesK(key, value)
}
func (responseHeader *_ResponseHeader) SetBytesV(key string, value []byte) {
	responseHeader.headre.SetBytesV(key, value)
}
func (responseHeader *_ResponseHeader) SetBytesKV(key, value []byte) {
	responseHeader.headre.SetBytesKV(key, value)
}
func (responseHeader *_ResponseHeader) SetCanonical(key, value []byte) {
	responseHeader.headre.SetCanonical(key, value)
}
func (responseHeader *_ResponseHeader) SetCookie(cookie Cookie) {
	responseHeader.headre.SetCookie(cookie.getFastHTTPCookie())
}
func (responseHeader *_ResponseHeader) DelClientCookie(key string) {
	responseHeader.headre.DelClientCookie(key)
}
func (responseHeader *_ResponseHeader) DelClientCookieBytes(key []byte) {
	responseHeader.headre.DelClientCookieBytes(key)
}
func (responseHeader *_ResponseHeader) DelCookieBytes(key []byte) {
	responseHeader.headre.DelCookieBytes(key)
}
func (responseHeader *_ResponseHeader) DelCookie(key string) {
	responseHeader.headre.DelCookie(key)
}
func (responseHeader *_ResponseHeader) DelAllCookies() {
	responseHeader.headre.DelAllCookies()
}
func (responseHeader *_ResponseHeader) Peek(key string) []byte {
	return responseHeader.headre.Peek(key)
}
func (responseHeader *_ResponseHeader) PeekBytes(key []byte) []byte {
	return responseHeader.headre.PeekBytes(key)
}
func (responseHeader *_ResponseHeader) Cookie(cookie Cookie) bool {
	return responseHeader.headre.Cookie(cookie.getFastHTTPCookie())
}
func (responseHeader *_ResponseHeader) Read(r *bufio.Reader) error {
	return responseHeader.headre.Read(r)
}
func (responseHeader *_ResponseHeader) Write(w *bufio.Writer) error {
	return responseHeader.headre.Write(w)
}
func (responseHeader *_ResponseHeader) WriteTo(w io.Writer) (int64, error) {
	return responseHeader.headre.WriteTo(w)
}
func (responseHeader *_ResponseHeader) Header() []byte {
	return responseHeader.headre.Header()
}
func (responseHeader *_ResponseHeader) String() string {
	return responseHeader.headre.String()
}
func (responseHeader *_ResponseHeader) AppendBytes(dst []byte) []byte {
	return responseHeader.headre.AppendBytes(dst)
}
