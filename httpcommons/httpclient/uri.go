package httpclient

import (
	"github.com/valyala/fasthttp"
	"io"
)

type _HttpURI struct {
	uri *fasthttp.URI
}

func (httpURI *_HttpURI) getFastHTTPURI() *fasthttp.URI {
	return httpURI.uri
}

func (httpURI *_HttpURI) CopyTo(dst URI) {
	httpURI.uri.CopyTo(dst.getFastHTTPURI())
}

func (httpURI *_HttpURI) Hash() []byte {
	return httpURI.uri.Hash()
}

func (httpURI *_HttpURI) SetHash(hash string) {
	httpURI.uri.SetHash(hash)
}

func (httpURI *_HttpURI) SetHashBytes(hash []byte) {
	httpURI.uri.SetHashBytes(hash)
}

func (httpURI *_HttpURI) QueryString() []byte {
	return httpURI.uri.QueryString()
}

func (httpURI *_HttpURI) SetQueryString(queryString string) {
	httpURI.uri.SetQueryString(queryString)
}

func (httpURI *_HttpURI) SetQueryStringBytes(queryString []byte) {
	httpURI.uri.SetQueryStringBytes(queryString)
}

func (httpURI *_HttpURI) Path() []byte {
	return httpURI.uri.Path()
}

func (httpURI *_HttpURI) SetPath(path string) {
	httpURI.uri.SetPath(path)
}

func (httpURI *_HttpURI) SetPathBytes(path []byte) {
	httpURI.uri.SetPathBytes(path)
}

func (httpURI *_HttpURI) PathOriginal() []byte {
	return httpURI.uri.PathOriginal()
}

func (httpURI *_HttpURI) Scheme() []byte {
	return httpURI.uri.Scheme()
}

func (httpURI *_HttpURI) SetScheme(scheme string) {
	httpURI.uri.SetScheme(scheme)
}

func (httpURI *_HttpURI) SetSchemeBytes(scheme []byte) {
	httpURI.uri.SetSchemeBytes(scheme)
}

func (httpURI *_HttpURI) Reset() {
	httpURI.uri.Reset()
}

func (httpURI *_HttpURI) Host() []byte {
	return httpURI.uri.Host()
}

func (httpURI *_HttpURI) SetHost(host string) {
	httpURI.uri.SetHost(host)
}

func (httpURI *_HttpURI) SetHostBytes(host []byte) {
	httpURI.uri.SetHostBytes(host)
}

func (httpURI *_HttpURI) Parse(host, uri []byte) {
	httpURI.uri.Parse(host, uri)
}

func (httpURI *_HttpURI) RequestURI() []byte {
	return httpURI.uri.RequestURI()
}

func (httpURI *_HttpURI) LastPathSegment() []byte {
	return httpURI.uri.LastPathSegment()
}

func (httpURI *_HttpURI) Update(newURI string) {
	httpURI.uri.Update(newURI)
}

func (httpURI *_HttpURI) UpdateBytes(newURI []byte) {
	httpURI.uri.UpdateBytes(newURI)
}

func (httpURI *_HttpURI) FullURI() []byte {
	return httpURI.uri.FullURI()
}

func (httpURI *_HttpURI) AppendBytes(dst []byte) []byte {
	return httpURI.uri.AppendBytes(dst)
}

func (httpURI *_HttpURI) WriteTo(w io.Writer) (int64, error) {
	return httpURI.uri.WriteTo(w)
}

func (httpURI *_HttpURI) QueryArgs() Args {
	return &_Args{
		args: httpURI.uri.QueryArgs(),
	}
}
