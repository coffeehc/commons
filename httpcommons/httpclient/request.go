package httpclient

import (
	"bufio"
	"github.com/valyala/fasthttp"
	"io"
	"mime/multipart"
)

type _Request struct {
	request *fasthttp.Request
}

func (request *_Request) getFastHTTPRequest() *fasthttp.Request {
	return request.request
}
func (request *_Request) SetHost(host string) {
	request.request.SetHost(host)
}
func (request *_Request) SetHostBytes(host []byte) {
	request.request.SetHostBytes(host)
}
func (request *_Request) Host() []byte {
	return request.request.Host()
}
func (request *_Request) SetRequestURI(requestURI string) {
	request.request.SetRequestURI(requestURI)
}
func (request *_Request) SetRequestURIBytes(requestURI []byte) {
	request.request.SetRequestURIBytes(requestURI)
}
func (request *_Request) RequestURI() []byte {
	return request.request.RequestURI()
}
func (request *_Request) ConnectionClose() bool {
	return request.request.ConnectionClose()
}
func (request *_Request) SetConnectionClose() {
	request.request.SetConnectionClose()
}
func (request *_Request) SetBodyStream(bodyStream io.Reader, bodySize int) {
	request.request.SetBodyStream(bodyStream, bodySize)
}
func (request *_Request) SetBodyStreamWriter(sw func(w *bufio.Writer)) {
	request.request.SetBodyStreamWriter(sw)
}
func (request *_Request) BodyWriter() io.Writer {
	return request.request.BodyWriter()
}
func (request *_Request) BodyGunzip() ([]byte, error) {
	return request.request.BodyGunzip()
}
func (request *_Request) BodyInflate() ([]byte, error) {
	return request.request.BodyInflate()
}
func (request *_Request) BodyWriteTo(w io.Writer) error {
	return request.request.BodyWriteTo(w)
}
func (request *_Request) ReleaseBody(size int) {
	request.request.ReleaseBody(size)
}
func (request *_Request) SwapBody(body []byte) []byte {
	return request.request.SwapBody(body)
}
func (request *_Request) Body() []byte {
	return request.request.Body()
}
func (request *_Request) AppendBody(p []byte) {
	request.request.AppendBody(p)
}
func (request *_Request) AppendBodyString(s string) {
	request.request.AppendBodyString(s)
}
func (request *_Request) SetBody(body []byte) {
	request.request.SetBody(body)
}
func (request *_Request) SetBodyString(body string) {
	request.request.SetBodyString(body)
}
func (request *_Request) ResetBody() {
	request.request.ResetBody()
}
func (request *_Request) CopyTo(dst Request) {
	request.request.CopyTo(dst.getFastHTTPRequest())
}
func (request *_Request) URI() URI {
	return &_HttpURI{
		uri: request.request.URI(),
	}
}
func (request *_Request) PostArgs() Args {
	return &_Args{
		args: request.request.PostArgs(),
	}
}
func (request *_Request) MultipartForm() (*multipart.Form, error) {
	return request.request.MultipartForm()
}
func (request *_Request) Reset() {
	request.request.Reset()
}
func (request *_Request) RemoveMultipartFormFiles() {
	request.request.RemoveMultipartFormFiles()
}
func (request *_Request) Read(r *bufio.Reader) error {
	return request.request.Read(r)
}
func (request *_Request) ReadLimitBody(r *bufio.Reader, maxBodySize int) error {
	return request.request.ReadLimitBody(r, maxBodySize)
}
func (request *_Request) MayContinue() bool {
	return request.request.MayContinue()
}
func (request *_Request) ContinueReadBody(r *bufio.Reader, maxBodySize int) error {
	return request.request.ContinueReadBody(r, maxBodySize)
}
func (request *_Request) WriteTo(w io.Writer) (int64, error) {
	return request.request.WriteTo(w)
}
func (request *_Request) Write(w *bufio.Writer) error {
	return request.request.Write(w)
}
func (request *_Request) String() string {
	return request.request.String()
}
