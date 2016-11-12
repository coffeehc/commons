package httpclient

import (
	"github.com/valyala/fasthttp"
	"io"
)

type _HttpUri struct {
	uri *fasthttp.URI
}

func (this *_HttpUri)getFastHttpUri() *fasthttp.URI {
	return this.uri
}

func (this *_HttpUri)CopyTo(dst HttpUri) {
	this.uri.CopyTo(dst)
}
func (this *_HttpUri)Hash() []byte {
	return this.uri.Hash()
}
func (this *_HttpUri)SetHash(hash string) {
	this.uri.SetHash(hash)
}
func (this *_HttpUri)SetHashBytes(hash []byte) {
	this.uri.SetHashBytes(hash)
}
func (this *_HttpUri)QueryString() []byte {
	return this.uri.QueryArgs()
}
func (this *_HttpUri)SetQueryString(queryString string) {
	this.uri.SetQueryString(queryString)
}
func (this *_HttpUri)SetQueryStringBytes(queryString []byte) {
	this.uri.SetQueryStringBytes(queryString)
}
func (this *_HttpUri)Path() []byte {
	return this.uri.Path()
}
func (this *_HttpUri)SetPath(path string) {
	this.uri.SetPath(path)
}
func (this *_HttpUri)SetPathBytes(path []byte) {
	this.uri.SetPathBytes(path)
}
func (this *_HttpUri)PathOriginal() []byte {
	return this.uri.PathOriginal()
}
func (this *_HttpUri)Scheme() []byte {
	return this.uri.Scheme()
}
func (this *_HttpUri)SetScheme(scheme string) {
	this.uri.SetScheme(scheme)
}
func (this *_HttpUri)SetSchemeBytes(scheme []byte) {
	this.uri.SetSchemeBytes(scheme)
}
func (this *_HttpUri)Reset() {
	this.uri.Reset()
}
func (this *_HttpUri)Host() []byte {
	return this.uri.Host()
}
func (this *_HttpUri)SetHost(host string) {
	this.uri.SetHost(host)
}
func (this *_HttpUri)SetHostBytes(host []byte) {
	this.uri.SetHostBytes(host)
}
func (this *_HttpUri)Parse(host, uri []byte) {
	this.uri.Parse(host, uri)
}
func (this *_HttpUri)RequestURI() []byte {
	return this.uri.RequestURI()
}
func (this *_HttpUri)LastPathSegment() []byte {
	return this.uri.LastPathSegment()
}
func (this *_HttpUri)Update(newURI string) {
	this.uri.Update(newURI)
}
func (this *_HttpUri)UpdateBytes(newURI []byte) {
	this.uri.UpdateBytes(newURI)
}
func (this *_HttpUri)FullURI() []byte {
	return this.uri.FullURI()
}
func (this *_HttpUri)AppendBytes(dst []byte) []byte {
	return this.uri.AppendBytes(dst)
}
func (this *_HttpUri)WriteTo(w io.Writer) (int64, error) {
	return this.uri.WriteTo(w)
}
func (this *_HttpUri)QueryArgs() HttpArgs {
	return this.uri.QueryArgs()
}
