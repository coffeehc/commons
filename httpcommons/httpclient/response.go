package httpclient

import (
	"bufio"
	"github.com/valyala/fasthttp"
	"io"
)

type _Response struct {
	response *fasthttp.Response
}

func (response *_Response) getFastHTTPResponse() *fasthttp.Response {
	return response.response
}

func (response *_Response) StatusCode() int {
	return response.response.StatusCode()
}
func (response *_Response) SetStatusCode(statusCode int) {
	response.response.SetStatusCode(statusCode)
}
func (response *_Response) ConnectionClose() bool {
	return response.response.ConnectionClose()
}
func (response *_Response) SetConnectionClose() {
	response.response.SetConnectionClose()
}
func (response *_Response) SendFile(path string) error {
	return response.response.SendFile(path)
}
func (response *_Response) SetBodyStream(bodyStream io.Reader, bodySize int) {
	response.response.SetBodyStream(bodyStream, bodySize)
}
func (response *_Response) IsBodyStream() bool {
	return response.response.IsBodyStream()
}
func (response *_Response) SetBodyStreamWriter(sw func(w *bufio.Writer)) {
	response.response.SetBodyStreamWriter(sw)
}
func (response *_Response) BodyWriter() io.Writer {
	return response.response.BodyWriter()
}
func (response *_Response) Body() []byte {
	return response.response.Body()
}
func (response *_Response) BodyGunzip() ([]byte, error) {
	return response.response.BodyGunzip()
}
func (response *_Response) BodyInflate() ([]byte, error) {
	return response.response.BodyInflate()
}
func (response *_Response) BodyWriteTo(w io.Writer) error {
	return response.response.BodyWriteTo(w)
}
func (response *_Response) AppendBody(p []byte) {
	response.response.AppendBody(p)
}
func (response *_Response) AppendBodyString(s string) {
	response.response.AppendBodyString(s)
}
func (response *_Response) SetBody(body []byte) {
	response.response.SetBody(body)
}
func (response *_Response) SetBodyString(body string) {
	response.response.SetBodyString(body)
}
func (response *_Response) ResetBody() {
	response.response.ResetBody()
}
func (response *_Response) ReleaseBody(size int) {
	response.response.ReleaseBody(size)
}
func (response *_Response) SwapBody(body []byte) []byte {
	return response.response.SwapBody(body)
}
func (response *_Response) CopyTo(dst Response) {
	response.response.CopyTo(dst.getFastHTTPResponse())
}
func (response *_Response) Reset() {
	response.response.Reset()
}
func (response *_Response) Read(r *bufio.Reader) error {
	return response.response.Read(r)
}
func (response *_Response) ReadLimitBody(r *bufio.Reader, maxBodySize int) error {
	return response.response.ReadLimitBody(r, maxBodySize)
}
func (response *_Response) WriteTo(w io.Writer) (int64, error) {
	return response.response.WriteTo(w)
}
func (response *_Response) WriteGzip(w *bufio.Writer) error {
	return response.response.WriteGzip(w)
}
func (response *_Response) WriteGzipLevel(w *bufio.Writer, level int) error {
	return response.response.WriteGzipLevel(w, level)
}
func (response *_Response) WriteDeflate(w *bufio.Writer) error {
	return response.response.WriteDeflate(w)
}
func (response *_Response) WriteDeflateLevel(w *bufio.Writer, level int) error {
	return response.response.WriteDeflateLevel(w, level)
}
func (response *_Response) Write(w *bufio.Writer) error {
	return response.response.Write(w)
}
func (response *_Response) String() string {
	return response.response.String()
}
