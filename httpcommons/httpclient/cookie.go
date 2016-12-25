package httpclient

import (
	"github.com/valyala/fasthttp"
	"io"
	"time"
)

type _Cookie struct {
	cookie *fasthttp.Cookie
}

func (cookie *_Cookie) getFastHTTPCookie() *fasthttp.Cookie {
	return cookie.cookie
}

func (cookie *_Cookie) CopyTo(src Cookie) {
	cookie.cookie.CopyTo(src.getFastHTTPCookie())
}
func (cookie *_Cookie) HTTPOnly() bool {
	return cookie.cookie.HTTPOnly()
}
func (cookie *_Cookie) SetHTTPOnly(httpOnly bool) {
	cookie.cookie.SetHTTPOnly(httpOnly)
}
func (cookie *_Cookie) Secure() bool {
	return cookie.cookie.Secure()
}
func (cookie *_Cookie) SetSecure(secure bool) {
	cookie.cookie.SetSecure(secure)
}
func (cookie *_Cookie) Path() []byte {
	return cookie.cookie.Path()
}
func (cookie *_Cookie) SetPath(path string) {
	cookie.cookie.SetPath(path)
}
func (cookie *_Cookie) SetPathBytes(path []byte) {
	cookie.cookie.SetPathBytes(path)
}
func (cookie *_Cookie) Domain() []byte {
	return cookie.cookie.Domain()
}
func (cookie *_Cookie) SetDomain(domain string) {
	cookie.cookie.SetDomain(domain)
}
func (cookie *_Cookie) SetDomainBytes(domain []byte) {
	cookie.cookie.SetDomainBytes(domain)
}
func (cookie *_Cookie) Expire() time.Time {
	return cookie.cookie.Expire()
}
func (cookie *_Cookie) SetExpire(expire time.Time) {
	cookie.cookie.SetExpire(expire)
}
func (cookie *_Cookie) Value() []byte {
	return cookie.cookie.Value()
}
func (cookie *_Cookie) SetValue(value string) {
	cookie.cookie.SetValue(value)
}
func (cookie *_Cookie) SetValueBytes(value []byte) {
	cookie.cookie.SetValueBytes(value)
}
func (cookie *_Cookie) Key() []byte {
	return cookie.cookie.Key()
}
func (cookie *_Cookie) SetKey(key string) {
	cookie.cookie.SetKey(key)
}
func (cookie *_Cookie) SetKeyBytes(key []byte) {
	cookie.cookie.SetKeyBytes(key)
}
func (cookie *_Cookie) Reset() {
	cookie.cookie.Reset()
}
func (cookie *_Cookie) AppendBytes(dst []byte) []byte {
	return cookie.cookie.AppendBytes(dst)
}
func (cookie *_Cookie) Cookie() []byte {
	return cookie.cookie.Cookie()
}
func (cookie *_Cookie) String() string {
	return cookie.cookie.String()
}
func (cookie *_Cookie) WriteTo(w io.Writer) (int64, error) {
	return cookie.cookie.WriteTo(w)
}
func (cookie *_Cookie) Parse(src string) error {
	return cookie.cookie.Parse(src)
}
func (cookie *_Cookie) ParseBytes(src []byte) error {
	return cookie.cookie.ParseBytes(src)
}
