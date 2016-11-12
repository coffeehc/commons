package httpclient

import (
	"github.com/valyala/fasthttp"
	"io"
	"bufio"
)

type _HttpResponse struct {
	response *fasthttp.Response
}

func (this *_HttpResponse)getFastHttpResponse() *fasthttp.Response {
	return this.response
}

func (this *_HttpResponse)StatusCode() int {
	this.response.StatusCode()
}
func (this *_HttpResponse)SetStatusCode(statusCode int) {
	this.response.SetStatusCode(statusCode)
}
func (this *_HttpResponse)ConnectionClose() bool {
	this.response.ConnectionClose()
}
func (this *_HttpResponse)SetConnectionClose() {
	this.response.SetConnectionClose()
}
func (this *_HttpResponse)SendFile(path string) error {
	this.response.SendFile(path)
}
func (this *_HttpResponse)SetBodyStream(bodyStream io.Reader, bodySize int) {
	this.response.SetBodyStream(bodyStream, bodySize)
}
func (this *_HttpResponse)IsBodyStream() bool {
	return this.response.IsBodyStream()
}
func (this *_HttpResponse)SetBodyStreamWriter(sw StreamWriter) {
	this.response.SetBodyStreamWriter(sw)
}
func (this *_HttpResponse)BodyWriter() io.Writer {
	return this.response.BodyWriter()
}
func (this *_HttpResponse)Body() []byte {
	return this.response.Body()
}
func (this *_HttpResponse)BodyGunzip() ([]byte, error) {
	return this.response.BodyGunzip()
}
func (this *_HttpResponse)BodyInflate() ([]byte, error) {
	return this.response.BodyInflate()
}
func (this *_HttpResponse)BodyWriteTo(w io.Writer) error {
	return this.response.BodyWriteTo(w)
}
func (this *_HttpResponse)AppendBody(p []byte) {
	this.response.AppendBody(p)
}
func (this *_HttpResponse)AppendBodyString(s string) {
	this.response.AppendBodyString(s)
}
func (this *_HttpResponse)SetBody(body []byte) {
	this.response.SetBody(body)
}
func (this *_HttpResponse)SetBodyString(body string) {
	this.response.SetBodyString(body)
}
func (this *_HttpResponse)ResetBody() {
	this.response.ResetBody()
}
func (this *_HttpResponse)ReleaseBody(size int) {
	this.response.ReleaseBody(size)
}
func (this *_HttpResponse)SwapBody(body []byte) []byte {
	return this.response.SwapBody(body)
}
func (this *_HttpResponse)CopyTo(dst HttpResponse) {
	this.response.CopyTo(dst)
}
func (this *_HttpResponse)Reset() {
	this.response.Reset()
}
func (this *_HttpResponse)Read(r *bufio.Reader) error {
	return this.response.Read(r)
}
func (this *_HttpResponse)ReadLimitBody(r *bufio.Reader, maxBodySize int) error {
	return this.response.ReadLimitBody(r, maxBodySize)
}
func (this *_HttpResponse)WriteTo(w io.Writer) (int64, error) {
	return this.response.WriteTo(w)
}
func (this *_HttpResponse)WriteGzip(w *bufio.Writer) error {
	return this.response.WriteGzip(w)
}
func (this *_HttpResponse)WriteGzipLevel(w *bufio.Writer, level int) error {
	return this.response.WriteGzipLevel(w, level)
}
func (this *_HttpResponse)WriteDeflate(w *bufio.Writer) error {
	return this.response.WriteDeflate(w)
}
func (this *_HttpResponse)WriteDeflateLevel(w *bufio.Writer, level int) error {
	return this.response.WriteDeflateLevel(w, level)
}
func (this *_HttpResponse)Write(w *bufio.Writer) error {
	return this.response.Write(w)
}
func (this *_HttpResponse)String() string {
	return this.response.String()
}
