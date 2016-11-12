package httpclient

import (
	"github.com/valyala/fasthttp"
	"time"
	"io"
)

type _HttpCookie struct {
	cookie *fasthttp.Cookie
}

func (this *_HttpCookie)getFastHttpCookie() *fasthttp.Cookie {
	return this.cookie
}

func (this *_HttpCookie)CopyTo(src HttpCookie) {
	this.cookie.CopyTo(src.getFastHttpCookie())
}
func (this *_HttpCookie)HTTPOnly() bool {
	return this.cookie.HTTPOnly()
}
func (this *_HttpCookie)SetHTTPOnly(httpOnly bool) {
	this.cookie.SetHTTPOnly(httpOnly)
}
func (this *_HttpCookie)Secure() bool {
	return this.cookie.Secure()
}
func (this *_HttpCookie)SetSecure(secure bool) {
	this.cookie.SetSecure(secure)
}
func (this *_HttpCookie)Path() []byte {
	return this.cookie.Path()
}
func (this *_HttpCookie)SetPath(path string) {
	this.cookie.SetPath(path)
}
func (this *_HttpCookie)SetPathBytes(path []byte) {
	this.cookie.SetPathBytes(path)
}
func (this *_HttpCookie)Domain() []byte {
	return this.cookie.Domain()
}
func (this *_HttpCookie)SetDomain(domain string) {
	this.cookie.SetDomain(domain)
}
func (this *_HttpCookie)SetDomainBytes(domain []byte) {
	this.cookie.SetDomainBytes(domain)
}
func (this *_HttpCookie)Expire() time.Time {
	return this.cookie.Expire()
}
func (this *_HttpCookie)SetExpire(expire time.Time) {
	this.cookie.SetExpire(expire)
}
func (this *_HttpCookie)Value() []byte {
	return this.cookie.Value()
}
func (this *_HttpCookie)SetValue(value string) {
	this.cookie.SetValue(value)
}
func (this *_HttpCookie)SetValueBytes(value []byte) {
	this.cookie.SetValueBytes(value)
}
func (this *_HttpCookie)Key() []byte {
	return this.cookie.Key()
}
func (this *_HttpCookie)SetKey(key string) {
	this.cookie.SetKey(key)
}
func (this *_HttpCookie)SetKeyBytes(key []byte) {
	this.cookie.SetKeyBytes(key)
}
func (this *_HttpCookie)Reset() {
	this.cookie.Reset()
}
func (this *_HttpCookie)AppendBytes(dst []byte) []byte {
	return this.cookie.AppendBytes(dst)
}
func (this *_HttpCookie)Cookie() []byte {
	return this.cookie.Cookie()
}
func (this *_HttpCookie)String() string {
	return this.cookie.String()
}
func (this *_HttpCookie)WriteTo(w io.Writer) (int64, error) {
	return this.cookie.WriteTo(w)
}
func (this *_HttpCookie)Parse(src string) error {
	return this.cookie.Parse(src)
}
func (this *_HttpCookie)ParseBytes(src []byte) error {
	return this.cookie.ParseBytes(src)
}
