package httpclient

import (
	"github.com/valyala/fasthttp"
	"io"
	"mime/multipart"
	"bufio"
)

type _HttpRequest struct {
	request *fasthttp.Request
}
func (this *_HttpRequest)getFastHttpRequest() *fasthttp.Request {
	return this.request
}
func (this *_HttpRequest)SetHost(host string) {
	this.request.SetHost(host)
}
func (this *_HttpRequest)SetHostBytes(host []byte) {
	this.request.SetHostBytes(host)
}
func (this *_HttpRequest)Host() []byte {
	return this.request.Host()
}
func (this *_HttpRequest)GetRequestURI(requestURI string) {
	this.request.RequestURI(requestURI)
}
func (this *_HttpRequest)SetRequestURI(requestURI string){
	this.request.SetRequestURI(requestURI)
}
func (this *_HttpRequest)SetRequestURIBytes(requestURI []byte) {
	this.request.SetRequestURIBytes(requestURI)
}
func (this *_HttpRequest)RequestURI() []byte {
	return this.request.RequestURI()
}
func (this *_HttpRequest)ConnectionClose() bool {
	return this.request.ConnectionClose()
}
func (this *_HttpRequest)SetConnectionClose() {
	this.request.SetConnectionClose()
}
func (this *_HttpRequest)SetBodyStream(bodyStream io.Reader, bodySize int) {
	this.request.SetBodyStream(bodyStream, bodySize)
}
func (this *_HttpRequest)SetBodyStreamWriter(sw StreamWriter) {
	this.request.SetBodyString(sw)
}
func (this *_HttpRequest)BodyWriter() io.Writer {
	this.request.BodyWriter()
}
func (this *_HttpRequest)BodyGunzip() ([]byte, error) {
	return this.request.BodyGunzip()
}
func (this *_HttpRequest)BodyInflate() ([]byte, error) {
	return this.request.BodyInflate()
}
func (this *_HttpRequest)BodyWriteTo(w io.Writer) error {
	return this.request.BodyWriteTo(w)
}
func (this *_HttpRequest)ReleaseBody(size int) {
	this.request.ReleaseBody()
}
func (this *_HttpRequest)SwapBody(body []byte) []byte {
	return this.request.SwapBody(body)
}
func (this *_HttpRequest)Body() []byte {
	return this.request.Body()
}
func (this *_HttpRequest)AppendBody(p []byte) {
	this.request.AppendBody(p)
}
func (this *_HttpRequest)AppendBodyString(s string) {
	this.request.AppendBodyString(s)
}
func (this *_HttpRequest)SetBody(body []byte) {
	this.request.SetBody(body)
}
func (this *_HttpRequest)SetBodyString(body string) {
	this.request.SetBodyString(body)
}
func (this *_HttpRequest)ResetBody() {
	this.request.ResetBody()
}
func (this *_HttpRequest)CopyTo(dst *HttpRequest) {
	this.request.CopyTo(dst)
}
func (this *_HttpRequest)URI() HttpUri {
	return this.request.URI()
}
func (this *_HttpRequest)PostArgs() HttpArgs {
	return this.request.PostArgs()
}
func (this *_HttpRequest)MultipartForm() (*multipart.Form, error) {
	return this.request.MultipartForm()
}
func (this *_HttpRequest)Reset() {
	this.request.Reset()
}
func (this *_HttpRequest)RemoveMultipartFormFiles() {
	this.request.RemoveMultipartFormFiles()
}
func (this *_HttpRequest)Read(r *bufio.Reader) error {
	this.request.Read(r)
}
func (this *_HttpRequest)ReadLimitBody(r *bufio.Reader, maxBodySize int) error {
	return this.request.ReadLimitBody(r, maxBodySize)
}
func (this *_HttpRequest)MayContinue() bool {
	return this.request.MayContinue()
}
func (this *_HttpRequest)ContinueReadBody(r *bufio.Reader, maxBodySize int) error {
	return this.request.ContinueReadBody(r, maxBodySize)
}
func (this *_HttpRequest)WriteTo(w io.Writer) (int64, error) {
	return this.request.WriteTo(w)
}
func (this *_HttpRequest)Write(w *bufio.Writer) error {
	return this.request.Write(w)
}
func (this *_HttpRequest)String() string {
	return this.request.String()
}
