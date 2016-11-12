package httpclient

import (
	"github.com/valyala/fasthttp"
	"bufio"
	"io"
	"time"
)

type _HttpResponseHeader struct {
	headre *fasthttp.ResponseHeader
}

func (this *_HttpResponseHeader)getFastHttpResponseHeader() *fasthttp.ResponseHeader {
	return this.headre
}

func (this *_HttpResponseHeader)SetContentRange(startPos, endPos, contentLength int){
	this.headre.SetContentRange(startPos,endPos,contentLength)
}
func (this *_HttpResponseHeader)StatusCode() int{
	return this.headre.StatusCode()
}
func (this *_HttpResponseHeader)SetStatusCode(statusCode int){
	this.headre.SetStatusCode(statusCode)
}
func (this *_HttpResponseHeader)SetLastModified(t time.Time){
	this.headre.SetLastModified(t)
}
func (this *_HttpResponseHeader)ConnectionClose() bool{
	return this.headre.ConnectionClose()
}
func (this *_HttpResponseHeader)SetConnectionClose(){
	this.headre.SetConnectionClose()
}
func (this *_HttpResponseHeader)ResetConnectionClose(){
	this.headre.ResetConnectionClose()
}
func (this *_HttpResponseHeader)ConnectionUpgrade() bool{
	return this.headre.ConnectionUpgrade()
}
func (this *_HttpResponseHeader)ContentLength() int{
	return this.headre.ContentLength()
}
func (this *_HttpResponseHeader)SetContentLength(contentLength int){
	this.headre.SetContentLength(contentLength)
}
func (this *_HttpResponseHeader)ContentType() []byte{
	return this.headre.ContentType()
}
func (this *_HttpResponseHeader)SetContentType(contentType string){
	this.headre.SetContentType(contentType)
}
func (this *_HttpResponseHeader)SetContentTypeBytes(contentType []byte){
	this.headre.SetContentTypeBytes(contentType)
}
func (this *_HttpResponseHeader)Server() []byte{
	return this.headre.Server()
}
func (this *_HttpResponseHeader)SetServer(server string){
	this.headre.SetServer(server)
}
func (this *_HttpResponseHeader)SetServerBytes(server []byte){
	this.headre.SetServerBytes(server)
}
func (this *_HttpResponseHeader)IsHTTP11() bool{
	return this.headre.IsHTTP11()
}
func (this *_HttpResponseHeader)Len() int{
	return this.headre.Len()
}
func (this *_HttpResponseHeader)DisableNormalizing(){
	this.headre.DisableNormalizing()
}
func (this *_HttpResponseHeader)Reset(){
	this.headre.Reset()
}
func (this *_HttpResponseHeader)CopyTo(dst HttpResponseHeader){
	this.headre.CopyTo(dst.getFastHttpResponseHeader())
}
func (this *_HttpResponseHeader)VisitAll(f func(key, value []byte)){
	this.headre.VisitAll(f)
}
func (this *_HttpResponseHeader)VisitAllCookie(f func(key, value []byte)){
	this.headre.VisitAllCookie(f)
}
func (this *_HttpResponseHeader)Del(key string){
	this.headre.Del(key)
}
func (this *_HttpResponseHeader)DelBytes(key []byte){
	this.headre.DelBytes(key)
}
func (this *_HttpResponseHeader)Add(key, value string){
	this.headre.Add(key,value)
}
func (this *_HttpResponseHeader)AddBytesK(key []byte, value string){
	this.headre.AddBytesK(key,value)
}
func (this *_HttpResponseHeader)AddBytesV(key string, value []byte){
	this.headre.AddBytesV(key,value)
}
func (this *_HttpResponseHeader)AddBytesKV(key, value []byte){
	this.headre.AddBytesKV(key,value)
}
func (this *_HttpResponseHeader)Set(key, value string){
	this.headre.Set(key,value)
}
func (this *_HttpResponseHeader)SetBytesK(key []byte, value string){
	this.headre.SetBytesK(key,value)
}
func (this *_HttpResponseHeader)SetBytesV(key string, value []byte){
	this.headre.SetBytesV(key,value)
}
func (this *_HttpResponseHeader)SetBytesKV(key, value []byte){
	this.headre.SetBytesKV(key,value)
}
func (this *_HttpResponseHeader)SetCanonical(key, value []byte){
	this.headre.SetCanonical(key,value)
}
func (this *_HttpResponseHeader)SetCookie(cookie HttpCookie){
	this.headre.SetCookie(cookie.getFastHttpCookie())
}
func (this *_HttpResponseHeader)DelClientCookie(key string){
	this.headre.DelClientCookie(key)
}
func (this *_HttpResponseHeader)DelClientCookieBytes(key []byte){
	this.headre.DelClientCookieBytes(key)
}
func (this *_HttpResponseHeader)DelCookieBytes(key []byte){
	this.headre.DelCookieBytes(key)
}
func (this *_HttpResponseHeader)DelCookie(key string){
	this.headre.DelCookie(key)
}
func (this *_HttpResponseHeader)DelAllCookies(){
	this.headre.DelAllCookies()
}
func (this *_HttpResponseHeader)Peek(key string) []byte{
	return this.headre.Peek(key)
}
func (this *_HttpResponseHeader)PeekBytes(key []byte) []byte{
	return this.headre.PeekBytes(key)
}
func (this *_HttpResponseHeader)Cookie(cookie HttpCookie) bool {
	return this.headre.Cookie(cookie.getFastHttpCookie())
}
func (this *_HttpResponseHeader)Read(r *bufio.Reader) error {
	return this.headre.Read(r)
}
func (this *_HttpResponseHeader)Write(w *bufio.Writer) error {
	return this.headre.Write(w)
}
func (this *_HttpResponseHeader)WriteTo(w io.Writer) (int64, error){
	return this.headre.WriteTo(w)
}
func (this *_HttpResponseHeader)Header() []byte {
	return this.headre.Header()
}
func (this *_HttpResponseHeader)String() string {
	return this.headre.String()
}
func (this *_HttpResponseHeader)AppendBytes(dst []byte) []byte {
	return this.headre.AppendBytes(dst)
}

